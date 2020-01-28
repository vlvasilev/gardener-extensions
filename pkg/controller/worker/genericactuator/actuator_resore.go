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

package genericactuator

import (
	"context"
	"encoding/json"
	"fmt"

	extensionscontroller "github.com/gardener/gardener-extensions/pkg/controller"
	workercontroller "github.com/gardener/gardener-extensions/pkg/controller/worker"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	machinev1alpha1 "github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (a *genericActuator) Restore(ctx context.Context, worker *extensionsv1alpha1.Worker, cluster *extensionscontroller.Cluster) error {
	workerDelegate, err := a.delegateFactory.WorkerDelegate(ctx, worker, cluster)
	if err != nil {
		return errors.Wrapf(err, "could not instantiate actuator context")
	}

	// Generate the desired machine deployments.
	wantedMachineDeployments, err := workerDelegate.GenerateMachineDeployments(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to generate the machine deployments")
	}

	// Get the list of all existing machine deployments.
	existingMachineDeployments := &machinev1alpha1.MachineDeploymentList{}
	if err := a.client.List(ctx, existingMachineDeployments, client.InNamespace(worker.Namespace)); err != nil {
		return err
	}

	// Parse the worker state to a separete machineDeployment states and attach them to
	// the corresponding machineDeployments which are to be deployed later
	a.logger.Info("Extracting the worker status.state", "worker", fmt.Sprintf("%s/%s", worker.Namespace, worker.Name))
	if err := a.addStateToMachineDeployment(ctx, worker, wantedMachineDeployments); err != nil {
		return err
	}

	wantedMachineDeployments = removeWantedDeploymentWithoutState(wantedMachineDeployments)

	// Delete the machine-controller-manager. During restoration MCM must not exists
	a.logger.Info("Deploying machine-control-manager", "worker", fmt.Sprintf("%s/%s", worker.Namespace, worker.Name))
	if err := a.deleteMachineControllerManager(ctx, worker); err != nil {
		return errors.Wrapf(err, "failed deleting machine-controller-manager")
	}

	// Do the actual restoration
	a.logger.Info("Deploying the machines and MacineSets", "worker", fmt.Sprintf("%s/%s", worker.Namespace, worker.Name))
	if err := a.deployMachineSetsAndMachines(ctx, worker, wantedMachineDeployments); err != nil {
		return errors.Wrapf(err, "failed restoration of the machineSet and the machines")
	}

	// Generate machine deployment configuration based on previously computed list of deployments and deploy them.
	a.logger.Info("Deploying the sored in worker status.state machine deployments", "worker", fmt.Sprintf("%s/%s", worker.Namespace, worker.Name))
	if err := a.deployMachineDeployments(ctx, cluster, worker, existingMachineDeployments, wantedMachineDeployments, workerDelegate.MachineClassKind(), true); err != nil {
		return errors.Wrapf(err, "failed to restore the machine deployment config")
	}

	return nil
}

func (a *genericActuator) addStateToMachineDeployment(ctx context.Context, worker *extensionsv1alpha1.Worker, wantedMachineDeployments workercontroller.MachineDeployments) error {
	// Make a deep copy to ensure that no one will change the state during the restoration
	workerCopy := worker.DeepCopy()

	if workerCopy.Status.State == nil || len(workerCopy.Status.State.Raw) <= 0 {
		return nil
	}

	// Parse the worker state to MachineDeploymentStates
	workerState := make(map[string]*workercontroller.MachineDeploymentState)
	if err := json.Unmarshal(workerCopy.Status.State.Raw, &workerState); err != nil {
		return err
	}

	// Attach the parsed MachineDeploymentStates to the wanted MachineDeployments
	for index, wantedMachineDeployment := range wantedMachineDeployments {
		wantedMachineDeployments[index].State = workerState[wantedMachineDeployment.Name]
	}

	return nil
}

func (a *genericActuator) deployMachineSetsAndMachines(ctx context.Context, worker *extensionsv1alpha1.Worker, wantedMachineDeployments workercontroller.MachineDeployments) error {
	for _, wantedMachineDeployment := range wantedMachineDeployments {
		if wantedMachineDeployment.State == nil || wantedMachineDeployment.State.MachineSet == nil {
			continue
		}

		// Extract the MachineSet
		machineSet := &machinev1alpha1.MachineSet{}
		if _, _, err := a.decoder.Decode(wantedMachineDeployment.State.MachineSet.Raw, nil, machineSet); err != nil {
			return err
		}

		// Create the MachineSet if not already exists. We do not care about the MachineSet status
		// because the MCM will update it
		if err := a.client.Create(ctx, machineSet); err != nil && !apierrors.IsAlreadyExists(err) {
			return err
		}

		// Deploy each machine owned by the MachineSet which was restored above
		for _, rawMachine := range wantedMachineDeployment.State.Machines {
			if rawMachine.Raw == nil {
				continue
			}
			// Unmarshal the Machine
			machine := &machinev1alpha1.Machine{}
			if _, _, err := a.decoder.Decode(rawMachine.Raw, nil, machine); err != nil {
				return err
			}

			// Create the machine if it not exists already
			err := a.client.Create(ctx, machine)
			if err != nil && !apierrors.IsAlreadyExists(err) {
				return err
			}

			// Attach the Shoot node to the Machine status
			node := machine.Status.Node
			if err := a.updateStatus(ctx, machine, apierrors.IsAlreadyExists(err), func() error {
				machine.Status.Node = node
				return nil
			}); err != nil {
				return err
			}
		}
	}

	return nil
}

// isAlreadyExists must be returned from from apierrors.IsAlreadyExists(err) where err commes from client.Create.
// This function is needed because wen you call client.Create and it successed if you immediately call TryUpdateStatus
// it will return an error stating that the Object does not exists.
// This bug comes form the fact that exponentialBackoff in TryUpdateStatus does not retry if client.Get returns error
// so the function exits immediately with error
func (a *genericActuator) updateStatus(ctx context.Context, obj runtime.Object, isAlreadyExists bool, transform func() error) error {
	if isAlreadyExists {
		return extensionscontroller.TryUpdateStatus(ctx, retry.DefaultBackoff, a.client, obj, transform)
	}
	return a.client.Status().Update(ctx, obj)
}

func removeWantedDeploymentWithoutState(wantedMachineDeployments workercontroller.MachineDeployments) workercontroller.MachineDeployments {
	var reducedMachineDeployments workercontroller.MachineDeployments
	for _, wantedMachineDeployment := range wantedMachineDeployments {
		if wantedMachineDeployment.State != nil {
			reducedMachineDeployments = append(reducedMachineDeployments, wantedMachineDeployment)
		}
	}
	return reducedMachineDeployments
}
