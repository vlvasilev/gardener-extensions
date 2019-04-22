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

package coreos

import (
	"github.com/gardener/gardener-extensions/pkg/controller/operatingsystemconfig"

	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// Type is the type of OperatingSystemConfigs the coreos actuator / predicate are built for.
const Type = "coreos"

var (
	// DefaultAddOptions are the default controller.Options for AddToManager.
	DefaultAddOptions = AddOptions{}
)

// AddOptions are the options for adding the controller to the manager.
type AddOptions struct {
	// Controller are the controller related options.
	Controller controller.Options
}

// AddToManagerWithOptions adds a controller with the given Options to the given manager.
// The opts.Reconciler is being set with a newly instantiated actuator.
func AddToManagerWithOptions(mgr manager.Manager, opts AddOptions) error {
	/*Add will add reconsiler struct which will hold the new Actuator. This struct has the Recosile function*/
	// 1.set the number of concurent reconsilation to one
	// 2.Inject the following functions in to the Manager
	// 	2.1 Inject dependencies into Reconciler
	// 		func (r *reconciler) InjectFunc(f inject.Func) error {
	// 	2.2 InjectClient injects the controller runtime client into the reconciler.
	// 		func (r *reconciler) InjectClient(client client.Client) error
	// 	2.3 func (r *reconciler) InjectScheme(scheme *runtime.Scheme) error
	// 3.Create controller with dependencies set get from the manager
	// 4.Add the controler (register it) to the manager
	// 5.Register Watch to watch for extensionsv1alpha1.OperatingSystemConfig resources related to this type
	// 6.Register Watch to watch for corev1.Secret resources related to this type*/
	return operatingsystemconfig.Add(mgr, operatingsystemconfig.AddArgs{
		Actuator:          NewActuator(),   // creates a new Actuator that updates the status of the handled OperatingSystemConfigs.
		ControllerOptions: opts.Controller, // zero recosilation and bear Recosiler intarface
		/*Check if event.Object.(*extensionsv1alpha1.OperatingSystemConfig).Spec.Type == typeName and if so return true*/
		/*Check if e.MetaOld.GetGeneration() != e.MetaNew.GetGeneration() and is so return True for Update*/
		Predicates: operatingsystemconfig.DefaultPredicates(Type),
	})
}

// AddToManager adds a controller with the default Options.
func AddToManager(mgr manager.Manager) error {
	return AddToManagerWithOptions(mgr, DefaultAddOptions)
}
