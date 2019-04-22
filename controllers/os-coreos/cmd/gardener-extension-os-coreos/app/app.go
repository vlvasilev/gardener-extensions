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

package app

import (
	"context"
	"os"

	"github.com/gardener/gardener-extensions/controllers/os-coreos/pkg/coreos"
	"github.com/gardener/gardener-extensions/pkg/controller"
	controllercmd "github.com/gardener/gardener-extensions/pkg/controller/cmd"
	"github.com/spf13/cobra"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// Name is the name of the CoreOS controller.
const Name = "os-coreos"

// NewControllerCommand creates a new CoreOS controller command.
func NewControllerCommand(ctx context.Context) *cobra.Command {
	var (
		/*IF RESTOptions.Kubeconfig is set retutns a client to authenticate gains kubeapiserver
		  ELSE IF global variable KUBECONFIG is set it used it to make the client.
		  ELSE IF Use the incluster configuration by searching for:
						tokenFile  = "/var/run/secrets/kubernetes.io/serviceaccount/token"
						rootCAFile = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
						$KUBERNETES_SERVICE_HOST
						$KUBERNETES_SERVICE_PORT
		  ELSE Use file under $HOME/.kube.conf
		*/
		restOpts = &controllercmd.RESTOptions{}
		mgrOpts  = &controllercmd.ManagerOptions{
			LeaderElection:          true,
			LeaderElectionID:        controllercmd.LeaderElectionNameID(Name), //os-coreos-leader-election
			LeaderElectionNamespace: os.Getenv("LEADER_ELECTION_NAMESPACE"),
		}
		ctrlOpts = &controllercmd.ControllerOptions{
			MaxConcurrentReconciles: 5,
		}

		aggOption = controllercmd.NewOptionAggregator(restOpts, mgrOpts, ctrlOpts)
	)

	cmd := &cobra.Command{
		Use: "os-coreos-controller-manager",

		Run: func(cmd *cobra.Command, args []string) {
			// set the config* pointer for restOpts, mgrOpts, ctrlOpts
			if err := aggOption.Complete(); err != nil {
				controllercmd.LogErrAndExit(err, "Error completing options")
			}

			mgr, err := manager.New(restOpts.Completed().Config, mgrOpts.Completed().Options())
			if err != nil {
				controllercmd.LogErrAndExit(err, "Could not instantiate manager")
			}

			if err := controller.AddToScheme(mgr.GetScheme()); err != nil {
				controllercmd.LogErrAndExit(err, "Could not update manager scheme")
			}

			ctrlOpts.Completed().Apply(&coreos.DefaultAddOptions.Controller)
			// AddToManagerWithOptions adds a controller with the given Options to the given manager.
			// The opts.Reconciler is being set with a newly instantiated actuator.
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
			if err := coreos.AddToManager(mgr); err != nil {
				controllercmd.LogErrAndExit(err, "Could not add controller to manager")
			}

			// Start starts all registered Controllers and blocks until the Stop channel is closed.
			if err := mgr.Start(ctx.Done()); err != nil {
				controllercmd.LogErrAndExit(err, "Error running manager")
			}
		},
	}

	aggOption.AddFlags(cmd.Flags())

	return cmd
}
