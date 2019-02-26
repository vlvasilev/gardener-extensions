// Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://wwr.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package worker

import (
	"context"

	"github.com/go-logr/logr"

	"github.com/gardener/gardener-extensions/pkg/util"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"

	"k8s.io/apimachinery/pkg/api/errors"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
	"sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	// FinalizerName is the worker controller finalizer.
	FinalizerName = "extensions.gardener.cloud/worker"
	name          = "worker-controller"
)

// NewReconciler creates a new reconcile.Reconciler that reconciles
// Worker resources of Gardener's `extensions.gardener.cloud` API group.
func NewReconciler(actuator Actuator) reconcile.Reconciler {
	return &reconciler{logger: log.Log.WithName(name), actuator: actuator}
}

// AddArgs are arguments for adding an worker controller to a manager.
type AddArgs struct {
	// Actuator is an worker actuator.
	Actuator Actuator
	// Type is the worker type the actuator supports.
	Type string
	// ControllerOptions are the controller options used for creating a controller.
	// The options.Reconciler is always overridden with a reconciler created from the
	// given actuator.
	ControllerOptions controller.Options
	// Predicates are the predicates to use.
	// If unset, GenerationChangedPredicate will be used.
	Predicates []predicate.Predicate
}

// Add creates a new Worker Controller and adds it to the Manager.
// and Start it when the Manager is Started.
func Add(mgr manager.Manager, args AddArgs) error {
	args.ControllerOptions.Reconciler = NewReconciler(args.Actuator)
	return add(mgr, args.Type, args.ControllerOptions, args.Predicates)
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, typeName string, options controller.Options, predicates []predicate.Predicate) error {
	ctrl, err := controller.New(name, mgr, options)
	if err != nil {
		return err
	}

	if predicates == nil {
		predicates = append(predicates, GenerationChangedPredicate())
	}
	predicates = append(predicates, TypePredicate(typeName))

	if err := ctrl.Watch(&source.Kind{Type: &extensionsv1alpha1.Worker{}}, &handler.EnqueueRequestForObject{}, predicates...); err != nil {
		return err
	}
	return nil
}

// reconciler reconciles Worker resources of Gardener's extensions.gardener.cloud` API group.
type reconciler struct {
	logger   logr.Logger
	actuator Actuator

	ctx    context.Context
	client client.Client
}

// InjectFunc enables dependency injection into the actuator.
func (r *reconciler) InjectFunc(f inject.Func) error {
	return f(r.actuator)
}

// InjectClient injects the controller runtime client into the reconciler.
func (r *reconciler) InjectClient(client client.Client) error {
	r.client = client
	return nil
}

// InjectStopChannel is an implementation for getting the respective stop channel managed by the controller-runtime.
func (r *reconciler) InjectStopChannel(stopCh <-chan struct{}) error {
	r.ctx = util.ContextFromStopChannel(stopCh)
	return nil
}

func (r *reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	worker := &extensionsv1alpha1.Worker{}
	err := r.client.Get(r.ctx, request.NamespacedName, worker)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if worker.DeletionTimestamp != nil {
		return r.delete(r.ctx, worker)
	}
	return r.reconcile(r.ctx, worker)
}
