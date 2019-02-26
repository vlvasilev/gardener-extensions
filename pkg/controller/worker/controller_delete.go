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

func (r *reconciler) delete(ctx context.Context, worker *extensionsv1alpha1.Worker) (reconcile.Result, error) {
	hasFinalizer, err := controllerutil.HasFinalizer(worker, FinalizerName)
	if err != nil {
		r.logger.Error(err, "Could not instantiate finalizer deletion")
		return reconcile.Result{}, err
	}

	if !hasFinalizer {
		r.logger.Info("Reconciling worker causes a no-op as there is no finalizer.", "worker", worker.Name)
		return reconcile.Result{}, nil
	}

	// TODO: implement delete logic here

	r.logger.Info("Worker deletion successful, removing finalizer.", "worker", worker.Name)
	if err := controllerutil.DeleteFinalizer(ctx, r.client, FinalizerName, worker); err != nil {
		r.logger.Error(err, "Error removing finalizer from worker", "worker", worker.Name)
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}
