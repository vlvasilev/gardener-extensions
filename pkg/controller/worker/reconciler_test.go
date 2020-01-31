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

package worker

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/gardener/gardener-extensions/pkg/controller"
	extensionscontroller "github.com/gardener/gardener-extensions/pkg/controller"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	extensionsclient "github.com/gardener/gardener/pkg/client/extensions/clientset/versioned/scheme"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func Test_reconciler_Reconcile(t *testing.T) {
	type fields struct {
		logger   logr.Logger
		actuator Actuator
		ctx      context.Context
		client   client.Client
	}
	type args struct {
		request reconcile.Request
	}
	logger := log.Log.WithName("Reconcile-Test-Controller")
	arguments := args{
		request: reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      "workerTestReconcile",
				Namespace: "test",
			},
		},
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    reconcile.Result
		wantErr bool
	}{
		{
			name: "test reconcile",
			fields: fields{
				logger:   logger,
				actuator: newFakeActuator(true, false, false, false),
				ctx:      context.TODO(),
				client: fake.NewFakeClientWithScheme(
					extensionsclient.Scheme,
					addOperationAnnotationToWorker(
						getWorker(),
						v1beta1constants.GardenerOperationReconcile),
					getCluster()),
			},
			args:    arguments,
			want:    reconcile.Result{},
			wantErr: false,
		},
		{
			name: "test after successful migrate",
			fields: fields{
				logger:   logger,
				actuator: newFakeActuator(false, false, false, false),
				ctx:      context.TODO(),
				client: fake.NewFakeClientWithScheme(
					extensionsclient.Scheme,
					addDeletionTimestampToWorker(
						addOperationAnnotationToWorker(
							addLastOperationToWorker(
								getWorker(),
								gardencorev1beta1.LastOperationTypeMigrate,
								gardencorev1beta1.LastOperationStateSucceeded,
								"Migrate worker"),
							v1beta1constants.GardenerOperationReconcile)),
					getCluster()),
			},
			args:    arguments,
			want:    reconcile.Result{},
			wantErr: false,
		},
		{
			name: "test migrate when operrationAnnotation Migrate occurs",
			fields: fields{
				logger:   logger,
				actuator: newFakeActuator(false, false, false, true),
				ctx:      context.TODO(),
				client: fake.NewFakeClientWithScheme(
					extensionsclient.Scheme,
					addOperationAnnotationToWorker(
						getWorker(),
						v1beta1constants.GardenerOperationMigrate),
					getCluster()),
			},
			args:    arguments,
			want:    reconcile.Result{},
			wantErr: false,
		},
		{
			name: "test error during migrate when operrationAnnotation Migrate occurs",
			fields: fields{
				logger:   logger,
				actuator: newFakeActuator(false, false, false, false),
				ctx:      context.TODO(),
				client: fake.NewFakeClientWithScheme(
					extensionsclient.Scheme,
					addOperationAnnotationToWorker(
						getWorker(),
						v1beta1constants.GardenerOperationMigrate),
					getCluster()),
			},
			args:    arguments,
			want:    reconcile.Result{},
			wantErr: true,
		},
		{
			name: "test Migrate after unssuccesful Migrate",
			fields: fields{
				logger:   logger,
				actuator: newFakeActuator(false, false, false, true),
				ctx:      context.TODO(),
				client: fake.NewFakeClientWithScheme(
					extensionsclient.Scheme,
					addLastOperationToWorker(
						getWorker(),
						gardencorev1beta1.LastOperationTypeMigrate,
						gardencorev1beta1.LastOperationStateFailed,
						"Migrate worker"),
					getCluster()),
			},
			args:    arguments,
			want:    reconcile.Result{},
			wantErr: false,
		},
		{
			name: "test error during Migrate after unssuccesful Migrate",
			fields: fields{
				logger:   logger,
				actuator: newFakeActuator(true, true, true, false),
				ctx:      context.TODO(),
				client: fake.NewFakeClientWithScheme(
					extensionsclient.Scheme,
					addLastOperationToWorker(
						getWorker(),
						gardencorev1beta1.LastOperationTypeMigrate,
						gardencorev1beta1.LastOperationStateFailed,
						"Migrate worker"),
					getCluster()),
			},
			args:    arguments,
			want:    reconcile.Result{},
			wantErr: true,
		},
		{
			name: "test Delete Worker",
			fields: fields{
				logger:   logger,
				actuator: newFakeActuator(false, true, false, false),
				ctx:      context.TODO(),
				client: fake.NewFakeClientWithScheme(
					extensionsclient.Scheme,
					addFinalizerToWorker(addDeletionTimestampToWorker(getWorker()), FinalizerName),
					getCluster()),
			},
			args:    arguments,
			want:    reconcile.Result{},
			wantErr: false,
		},
		{
			name: "test error when Delete Worker",
			fields: fields{
				logger:   logger,
				actuator: newFakeActuator(true, false, true, true),
				ctx:      context.TODO(),
				client: fake.NewFakeClientWithScheme(
					extensionsclient.Scheme,
					addFinalizerToWorker(addDeletionTimestampToWorker(getWorker()), FinalizerName),
					getCluster()),
			},
			args:    arguments,
			want:    reconcile.Result{},
			wantErr: true,
		},
		{
			name: "test restore when operrationAnnotation Restore occurs",
			fields: fields{
				logger:   logger,
				actuator: newFakeActuator(false, false, true, false),
				ctx:      context.TODO(),
				client: fake.NewFakeClientWithScheme(
					extensionsclient.Scheme,
					addOperationAnnotationToWorker(
						getWorker(),
						v1beta1constants.GardenerOperationRestore),
					getCluster()),
			},
			args:    arguments,
			want:    reconcile.Result{},
			wantErr: false,
		},
		{
			name: "test error restore when operrationAnnotation Restore occurs",
			fields: fields{
				logger:   logger,
				actuator: newFakeActuator(true, true, false, true),
				ctx:      context.TODO(),
				client: fake.NewFakeClientWithScheme(
					extensionsclient.Scheme,
					addOperationAnnotationToWorker(
						getWorker(),
						v1beta1constants.GardenerOperationRestore),
					getCluster()),
			},
			args:    arguments,
			want:    reconcile.Result{},
			wantErr: true,
		},
		{
			name: "test reconcile after failed reconcilation",
			fields: fields{
				logger:   logger,
				actuator: newFakeActuator(true, false, false, false),
				ctx:      context.TODO(),
				client: fake.NewFakeClientWithScheme(
					extensionsclient.Scheme,
					addLastOperationToWorker(
						getWorker(),
						gardencorev1beta1.LastOperationTypeReconcile,
						gardencorev1beta1.LastOperationStateFailed,
						"Reconcile worker"),
					getCluster()),
			},
			args:    arguments,
			want:    reconcile.Result{},
			wantErr: false,
		},
		{
			name: "test reconcile after successful restoration reconcilation",
			fields: fields{
				logger:   logger,
				actuator: newFakeActuator(true, false, false, false),
				ctx:      context.TODO(),
				client: fake.NewFakeClientWithScheme(
					extensionsclient.Scheme,
					addLastOperationToWorker(
						getWorker(),
						gardencorev1beta1.LastOperationTypeReconcile,
						gardencorev1beta1.LastOperationStateProcessing,
						"Processs worker reconcilation"),
					getCluster()),
			},
			args:    arguments,
			want:    reconcile.Result{},
			wantErr: false,
		},
		{
			name: "test error while reconciliation after failed reconcilation",
			fields: fields{
				logger:   logger,
				actuator: newFakeActuator(false, true, true, true),
				ctx:      context.TODO(),
				client: fake.NewFakeClientWithScheme(
					extensionsclient.Scheme,
					addLastOperationToWorker(
						getWorker(),
						gardencorev1beta1.LastOperationTypeReconcile,
						gardencorev1beta1.LastOperationStateFailed,
						"Reconcile worker"),
					getCluster()),
			},
			args:    arguments,
			want:    reconcile.Result{},
			wantErr: true,
		},
		{
			name: "test error while reconciliation after successful restoration reconcilation",
			fields: fields{
				logger:   logger,
				actuator: newFakeActuator(false, true, true, true),
				ctx:      context.TODO(),
				client: fake.NewFakeClientWithScheme(
					extensionsclient.Scheme,
					addLastOperationToWorker(
						getWorker(),
						gardencorev1beta1.LastOperationTypeReconcile,
						gardencorev1beta1.LastOperationStateProcessing,
						"Processs worker reconcilation"),
					getCluster()),
			},
			args:    arguments,
			want:    reconcile.Result{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &reconciler{
				logger:   tt.fields.logger,
				actuator: tt.fields.actuator,
				ctx:      tt.fields.ctx,
				client:   tt.fields.client,
			}
			got, err := r.Reconcile(tt.args.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("reconciler.Reconcile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("reconciler.Reconcile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func getWorker() *extensionsv1alpha1.Worker {
	return &extensionsv1alpha1.Worker{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Worker",
			APIVersion: "extensions.gardener.cloud/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "workerTestReconcile",
			Namespace: "test",
		},
		Spec: extensionsv1alpha1.WorkerSpec{},
	}
}

func addOperationAnnotationToWorker(worker *extensionsv1alpha1.Worker, annotation string) *extensionsv1alpha1.Worker {
	worker.Annotations = make(map[string]string)
	worker.Annotations[v1beta1constants.GardenerOperation] = annotation
	return worker
}

func addLastOperationToWorker(worker *extensionsv1alpha1.Worker, lastOperationType gardencorev1beta1.LastOperationType, lastOperationState gardencorev1beta1.LastOperationState, description string) *extensionsv1alpha1.Worker {
	worker.Status.LastOperation = extensionscontroller.LastOperation(lastOperationType, lastOperationState, 1, description)
	return worker
}

func addDeletionTimestampToWorker(worker *extensionsv1alpha1.Worker) *extensionsv1alpha1.Worker {
	worker.DeletionTimestamp = &metav1.Time{Time: time.Now()}
	return worker
}

func addFinalizerToWorker(worker *extensionsv1alpha1.Worker, finalizer string) *extensionsv1alpha1.Worker {
	worker.Finalizers = append(worker.Finalizers, finalizer)
	return worker
}

func getCluster() *extensionsv1alpha1.Cluster {
	return &extensionsv1alpha1.Cluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Cluster",
			APIVersion: "extensions.gardener.cloud/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
	}
}

// func addUpdateWorkerReactor(fakeClient *fake.Clientset) *fake.Clientset {
// 	fakeClient.AddReactor("update", "worker", func(action testaction.Action) (handled bool, ret runtime.Object, err error) {
// 		obj := action.(testaction.UpdateAction).GetObject().(*extensionsv1alpha1.Worker)
// 		return true, obj, nil
// 	})
// 	return fakeClient
// }

type fakeActuator struct {
	reconcile bool
	delete    bool
	restore   bool
	migrate   bool
}

func newFakeActuator(reconcile, delete, restore, migrate bool) Actuator {
	return &fakeActuator{
		reconcile: reconcile,
		delete:    delete,
		restore:   restore,
		migrate:   migrate,
	}
}

func (a *fakeActuator) Reconcile(ctx context.Context, worker *extensionsv1alpha1.Worker, cluster *controller.Cluster) error {
	if a.reconcile {
		return nil
	}
	return fmt.Errorf("Wrong function call: actuator Reconcile")
}

func (a *fakeActuator) Delete(ctx context.Context, worker *extensionsv1alpha1.Worker, cluster *controller.Cluster) error {
	if a.delete {
		return nil
	}
	return fmt.Errorf("Wrong function call: actuator Delete")
}

func (a *fakeActuator) Restore(ctx context.Context, worker *extensionsv1alpha1.Worker, cluster *controller.Cluster) error {
	if a.restore {
		return nil
	}
	return fmt.Errorf("Wrong function call: actuator Restore")
}

func (a *fakeActuator) Migrate(ctx context.Context, worker *extensionsv1alpha1.Worker, cluster *controller.Cluster) error {
	if a.migrate {
		return nil
	}
	return fmt.Errorf("Wrong function call: actuator Migrate")
}
