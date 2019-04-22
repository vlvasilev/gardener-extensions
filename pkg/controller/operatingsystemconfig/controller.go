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

package operatingsystemconfig

import (
	extensionscontroller "github.com/gardener/gardener-extensions/pkg/controller"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"

	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"

	corev1 "k8s.io/api/core/v1"
)

const (
	// FinalizerName is the name of the finalizer written by this controller.
	FinalizerName = "extensions.gardener.cloud/operatingsystemconfigs"

	name = "operatingsystemconfig-controller"
)

// AddArgs are arguments for adding an operatingsystemconfig controller to a manager.
type AddArgs struct {
	// Actuator is an operatingsystemconfig actuator.
	Actuator Actuator
	// ControllerOptions are the controller options used for creating a controller.
	// The options.Reconciler is always overridden with a reconciler created from the
	// given actuator.
	ControllerOptions controller.Options
	// Predicates are the predicates to use.
	// If unset, GenerationChangedPredicate will be used.
	Predicates []predicate.Predicate
}

// Add adds an operatingsystemconfig controller to the given manager using the given AddArgs.
func Add(mgr manager.Manager, args AddArgs) error {
	//reconciler reconciles OperatingSystemConfig resources of Gardener's `extensions.gardener.cloud` API group.
	args.ControllerOptions.Reconciler = NewReconciler(args.Actuator)
	return add(mgr, args.ControllerOptions, args.Predicates)
}

// DefaultPredicates returns the default predicates for an operatingsystemconfig reconciler.
func DefaultPredicates(typeName string) []predicate.Predicate {
	return []predicate.Predicate{
		/*Check if event.Object.(*extensionsv1alpha1.OperatingSystemConfig).Spec.Type == typeName and if so return true*/
		TypePredicate(typeName),
		/*Check if e.MetaOld.GetGeneration() != e.MetaNew.GetGeneration() and is so return True for Update*/
		extensionscontroller.GenerationChangedPredicate(),
	}
}

/*
  1.set the number of concurent reconsilation to one
  2.Inject the following functions in to the Manager
		2.1 Inject dependencies into Reconciler
			func (r *reconciler) InjectFunc(f inject.Func) error {
		2.2 InjectClient injects the controller runtime client into the reconciler.
			func (r *reconciler) InjectClient(client client.Client) error
		2.3 func (r *reconciler) InjectScheme(scheme *runtime.Scheme) error
  3.Create controller with dependencies set get from the manager
  4.Add the controler (register it) to the manager
  5.Register Watch in the controler to watch for extensionsv1alpha1.OperatingSystemConfig resources related to this type
  6.Register Watch in the controler to watch for corev1.Secret resources related to this type*/
func add(mgr manager.Manager, options controller.Options, predicates []predicate.Predicate) error {
	ctrl, err := controller.New("operatingsystemconfig-controller", mgr, options)
	if err != nil {
		return err
	}

	if err := ctrl.Watch(&source.Kind{Type: &extensionsv1alpha1.OperatingSystemConfig{}}, &handler.EnqueueRequestForObject{}, predicates...); err != nil {
		return err
	}

	if err := ctrl.Watch(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestsFromMapFunc{ToRequests: SecretToOSCMapper(mgr.GetClient(), predicates)}); err != nil {
		return err
	}

	return nil
}
