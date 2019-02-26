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

	controllerutil "github.com/gardener/gardener-extensions/pkg/controller"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (r *reconciler) reconcile(ctx context.Context, worker *extensionsv1alpha1.Worker) (reconcile.Result, error) {
	if err := controllerutil.EnsureFinalizer(ctx, r.client, FinalizerName, worker); err != nil {
		return reconcile.Result{}, err
	}

	// TODO: implement reconcile logic here
	r.logger.Info("Reconciling worker triggers idempotent update.", "worker", worker.Name)

	_, _, _ = r.actuator.GetMachineClassInfo()

	// return controllerutil.ReconcileErr(err)
	return reconcile.Result{}, nil
}
