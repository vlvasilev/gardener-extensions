// Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package backupbucket

import (
	"context"
	"fmt"

	extensionscontroller "github.com/gardener/gardener-extensions/pkg/controller"
	"github.com/gardener/gardener-extensions/pkg/util"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardencorev1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	gardencorev1beta1helper "github.com/gardener/gardener/pkg/apis/core/v1beta1/helper"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"

	"github.com/go-logr/logr"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
)

const (
	// EventBackupBucketReconciliation an event reason to describe backup entry reconciliation.
	EventBackupBucketReconciliation string = "BackupBucketReconciliation"
	// EventBackupBucketDeletion an event reason to describe backup entry deletion.
	EventBackupBucketDeletion string = "BackupBucketDeletion"
)

type reconciler struct {
	logger   logr.Logger
	actuator Actuator

	ctx      context.Context
	client   client.Client
	recorder record.EventRecorder
}

// NewReconciler creates a new reconcile.Reconciler that reconciles
// backupbucket resources of Gardener's `extensions.gardener.cloud` API group.
func NewReconciler(mgr manager.Manager, actuator Actuator) reconcile.Reconciler {
	return extensionscontroller.OperationAnnotationWrapper(
		&extensionsv1alpha1.BackupBucket{},
		&reconciler{
			logger:   log.Log.WithName(ControllerName),
			actuator: actuator,
			recorder: mgr.GetEventRecorderFor(ControllerName),
		})
}

func (r *reconciler) InjectFunc(f inject.Func) error {
	return f(r.actuator)
}

func (r *reconciler) InjectClient(client client.Client) error {
	r.client = client
	return nil
}

func (r *reconciler) InjectStopChannel(stopCh <-chan struct{}) error {
	r.ctx = util.ContextFromStopChannel(stopCh)
	return nil
}

func (r *reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	bb := &extensionsv1alpha1.BackupBucket{}
	if err := r.client.Get(r.ctx, request.NamespacedName, bb); err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// At the moment migration is not need for this controller
	if opAnnotation, ok := bb.Annotations[gardencorev1beta1constants.GardenerOperation]; ok && opAnnotation == gardencorev1beta1constants.GardenerOperationMigrate {
		return reconcile.Result{}, extensionscontroller.RemoveAnnotation(r.ctx, r.client, bb, gardencorev1beta1constants.GardenerOperation)
	}

	if bb.DeletionTimestamp != nil {
		return r.delete(r.ctx, bb)
	}
	return r.reconcile(r.ctx, bb)
}

func (r *reconciler) reconcile(ctx context.Context, bb *extensionsv1alpha1.BackupBucket) (reconcile.Result, error) {
	if err := extensionscontroller.EnsureFinalizer(ctx, r.client, FinalizerName, bb); err != nil {
		r.logger.Info("failed to ensure finalizer on backup bucket, %v", err)
		return reconcile.Result{}, err
	}

	operationType := gardencorev1beta1helper.ComputeOperationType(bb.ObjectMeta, bb.Status.LastOperation)
	if err := r.updateStatusProcessing(ctx, bb, operationType, "Reconciling the backupbucket"); err != nil {
		return reconcile.Result{}, err
	}

	secret, err := extensionscontroller.GetSecretByReference(ctx, r.client, &bb.Spec.SecretRef)
	if err != nil {
		r.logger.Info("failed to get backup bucket secret, %v", err)
		return reconcile.Result{}, err
	}
	if err := extensionscontroller.EnsureFinalizer(ctx, r.client, FinalizerName, secret); err != nil {
		r.logger.Info("failed to ensure finalizer on bucket secret, %v", err)
		return reconcile.Result{}, err
	}

	r.logger.Info("Starting the reconciliation of backupbucket", "backupbucket", bb.Name)
	r.recorder.Event(bb, corev1.EventTypeNormal, EventBackupBucketReconciliation, "Reconciling the backupbucket")
	if err := r.actuator.Reconcile(ctx, bb); err != nil {
		msg := "Error reconciling backupbucket"
		_ = r.updateStatusError(ctx, extensionscontroller.ReconcileErrCauseOrErr(err), bb, operationType, msg)
		r.logger.Error(err, msg, "backupbucket", bb.Name)
		return extensionscontroller.ReconcileErr(err)
	}

	if err := extensionscontroller.RemoveAnnotation(r.ctx, r.client, bb, gardencorev1beta1constants.GardenerOperation); err != nil {
		r.logger.Info("failed to remove operationAnnotation, %v", err)
		return reconcile.Result{}, err
	}

	msg := "Successfully reconciled backupbucket"
	r.logger.Info(msg, "backupbucket", bb.Name)
	r.recorder.Event(bb, corev1.EventTypeNormal, EventBackupBucketReconciliation, msg)
	if err := r.updateStatusSuccess(ctx, bb, operationType, msg); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *reconciler) delete(ctx context.Context, bb *extensionsv1alpha1.BackupBucket) (reconcile.Result, error) {
	hasFinalizer, err := extensionscontroller.HasFinalizer(bb, FinalizerName)
	if err != nil {
		r.logger.Error(err, "Could not instantiate finalizer deletion")
		return reconcile.Result{}, err
	}
	if !hasFinalizer {
		r.logger.Info("Deleting backupbucket causes a no-op as there is no finalizer.", "backupbucket", bb.Name)
		return reconcile.Result{}, nil
	}

	operationType := gardencorev1beta1helper.ComputeOperationType(bb.ObjectMeta, bb.Status.LastOperation)
	if err := r.updateStatusProcessing(ctx, bb, operationType, "Deleting the backupbucket"); err != nil {
		return reconcile.Result{}, err
	}

	r.logger.Info("Starting the deletion of backupbucket", "backupbucket", bb.Name)
	r.recorder.Event(bb, corev1.EventTypeNormal, EventBackupBucketDeletion, "Deleting the backupbucket")
	if err := r.actuator.Delete(r.ctx, bb); err != nil {
		msg := "Error deleting backupbucket"
		r.recorder.Eventf(bb, corev1.EventTypeWarning, EventBackupBucketDeletion, "%s: %+v", msg, err)
		_ = r.updateStatusError(ctx, extensionscontroller.ReconcileErrCauseOrErr(err), bb, operationType, msg)
		r.logger.Error(err, msg, "backupbucket", bb.Name)
		return extensionscontroller.ReconcileErr(err)
	}

	msg := "Successfully deleted backupbucket"
	r.logger.Info(msg, "backupbucket", bb.Name)
	r.recorder.Event(bb, corev1.EventTypeNormal, EventBackupBucketDeletion, msg)
	if err := r.updateStatusSuccess(ctx, bb, operationType, msg); err != nil {
		return reconcile.Result{}, err
	}

	secret, err := extensionscontroller.GetSecretByReference(ctx, r.client, &bb.Spec.SecretRef)
	if err != nil {
		r.logger.Info("failed to get backup bucket secret, %v", err)
		return reconcile.Result{}, err
	}
	if err := extensionscontroller.DeleteFinalizer(ctx, r.client, FinalizerName, secret); err != nil {
		r.logger.Info("failed to remove finalizer on bucket secret, %v", err)
		return reconcile.Result{}, err
	}

	r.logger.Info("Removing finalizer.", "backupbucket", bb.Name)
	if err := extensionscontroller.DeleteFinalizer(ctx, r.client, FinalizerName, bb); err != nil {
		r.logger.Error(err, "Error removing finalizer from backupbucket", "backupbucket", bb.Name)
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *reconciler) updateStatusProcessing(ctx context.Context, bb *extensionsv1alpha1.BackupBucket, lastOperationType gardencorev1beta1.LastOperationType, description string) error {
	return extensionscontroller.TryUpdateStatus(ctx, retry.DefaultBackoff, r.client, bb, func() error {
		bb.Status.LastOperation = extensionscontroller.LastOperation(lastOperationType, gardencorev1beta1.LastOperationStateProcessing, 1, description)
		return nil
	})
}

func (r *reconciler) updateStatusError(ctx context.Context, err error, bb *extensionsv1alpha1.BackupBucket, lastOperationType gardencorev1beta1.LastOperationType, description string) error {
	return extensionscontroller.TryUpdateStatus(ctx, retry.DefaultBackoff, r.client, bb, func() error {
		bb.Status.ObservedGeneration = bb.Generation
		bb.Status.LastOperation, bb.Status.LastError = extensionscontroller.ReconcileError(lastOperationType, gardencorev1beta1helper.FormatLastErrDescription(fmt.Errorf("%s: %v", description, err)), 50, gardencorev1beta1helper.ExtractErrorCodes(err)...)
		return nil
	})
}

func (r *reconciler) updateStatusSuccess(ctx context.Context, bb *extensionsv1alpha1.BackupBucket, lastOperationType gardencorev1beta1.LastOperationType, description string) error {
	return extensionscontroller.TryUpdateStatus(ctx, retry.DefaultBackoff, r.client, bb, func() error {
		bb.Status.ObservedGeneration = bb.Generation
		bb.Status.LastOperation, bb.Status.LastError = extensionscontroller.ReconcileSucceeded(lastOperationType, description)
		return nil
	})
}
