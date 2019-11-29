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
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	machinev1alpha1 "github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (a *genericActuator) restore(ctx context.Context, worker *extensionsv1alpha1.Worker, existingMachineDeployments *machinev1alpha1.MachineDeploymentList, wantedMachineDeployments workercontroller.MachineDeployments) error {

	if worker.Annotations[v1beta1constants.GardenerOperation] != v1beta1constants.GardenerOperationRestore {
		return nil
	}
	fmt.Println("DEBUG1: found a restore operation annotation and will remove it.")
	// remove operation annotation 'restore'
	withOpAnnotationRestore := worker.DeepCopyObject()
	delete(worker.Annotations, v1beta1constants.GardenerOperation)
	if err := a.client.Patch(ctx, worker, client.MergeFrom(withOpAnnotationRestore)); err != nil {
		fmt.Println("Failt to remove operation annotation")
		return err
	}

	// Parse the worker state to a separete machineDeployment state and attach it to
	// the corresponding machineDeployments
	if err := a.addStateToMachineDeployment(ctx, worker, wantedMachineDeployments); err != nil {
		return err
	}

	// Do the actual restoration
	return a.deployMachineSetsAndMachines(ctx, worker, existingMachineDeployments, wantedMachineDeployments)
}

func (a *genericActuator) addStateToMachineDeployment(ctx context.Context, worker *extensionsv1alpha1.Worker, wantedMachineDeployments workercontroller.MachineDeployments) error {
	workerCopy := worker.DeepCopy()
	fmt.Println("DEBUG2: Start to parese the wrker state")
	if workerCopy.Status.State == nil || len(workerCopy.Status.State.Raw) <= 0 {
		return nil
	}
	workerState := make(map[string]*workercontroller.MachineDeploymentState)

	if err := json.Unmarshal(workerCopy.Status.State.Raw, &workerState); err != nil {
		return err
	}

	for index, wantedMachineDeployment := range wantedMachineDeployments {
		wantedMachineDeployments[index].State = workerState[wantedMachineDeployment.Name]
		fmt.Printf("DEBUG3: For wanted machine deployment %s we found state %v!\n", wantedMachineDeployment.Name, wantedMachineDeployments[index].State)
	}
	return nil
}

func (a *genericActuator) deployMachineSetsAndMachines(ctx context.Context, worker *extensionsv1alpha1.Worker, existingMachineDeployments *machinev1alpha1.MachineDeploymentList, wantedMachineDeployments workercontroller.MachineDeployments) error {
	existingMachineDeploymentNameSet := make(map[string]struct{})

	for _, existingMachineDeployment := range existingMachineDeployments.Items {
		existingMachineDeploymentNameSet[existingMachineDeployment.Name] = struct{}{}
	}

	for _, wantedMachineDeployment := range wantedMachineDeployments {
		fmt.Printf("DEBUG5: start to restore deployment %s\n", wantedMachineDeployment.Name)
		// We restore machineSet and machines only for missing machineDeployments
		if _, ok := existingMachineDeploymentNameSet[wantedMachineDeployment.Name]; ok ||
			wantedMachineDeployment.State == nil || wantedMachineDeployment.State.MachineSet == nil {
			fmt.Println("OK: ", ok)
			fmt.Println("wantedMachineDeployment.State == nil: ", wantedMachineDeployment.State == nil)
			fmt.Println("wantedMachineDeployment.State.MachineSet == nil: ", wantedMachineDeployment.State.MachineSet == nil)
			continue
		}

		machineSet := &machinev1alpha1.MachineSet{}
		if _, _, err := a.decoder.Decode(wantedMachineDeployment.State.MachineSet.Raw, nil, machineSet); err != nil {
			return err
		}

		if err := a.client.Create(ctx, machineSet); err != nil && !apierrors.IsAlreadyExists(err) {
			return err
		}
		fmt.Printf("DEBUG6: Restore machine set %s\n", machineSet.Name)

		for _, rawMachine := range wantedMachineDeployment.State.Machines {
			if rawMachine.Raw == nil {
				continue
			}
			machine := &machinev1alpha1.Machine{}
			if _, _, err := a.decoder.Decode(rawMachine.Raw, nil, machine); err != nil {
				return err
			}
			err := a.client.Create(ctx, machine)
			if err != nil && !apierrors.IsAlreadyExists(err) {
				return err
			}
			node := machine.Status.Node
			if err := a.update(ctx, machine, apierrors.IsAlreadyExists(err), func() error {
				machine.Status.Node = node
				return nil
			}); err != nil {
				return err
			}
			fmt.Printf("DEBUG7: Restore machine  %s\n", machine.Name)
		}
	}

	return nil
}

func (a *genericActuator) update(ctx context.Context, obj runtime.Object, isAlreadyExists bool, transform func() error) error {
	if isAlreadyExists {
		return extensionscontroller.TryUpdateStatus(ctx, retry.DefaultBackoff, a.client, obj, transform)
	}
	return a.client.Status().Update(ctx, obj)
}
