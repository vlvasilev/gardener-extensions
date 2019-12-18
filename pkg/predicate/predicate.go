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

package predicate

import (
	"errors"

	"github.com/gardener/gardener-extensions/pkg/controller"
	extensionsevent "github.com/gardener/gardener-extensions/pkg/event"
	extensionsinject "github.com/gardener/gardener-extensions/pkg/inject"
	"github.com/gardener/gardener/pkg/api/extensions"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"

	"github.com/go-logr/logr"

	gardencorev1alpha1 "github.com/gardener/gardener/pkg/apis/core/v1alpha1"
	v1alpha1constants "github.com/gardener/gardener/pkg/apis/core/v1alpha1/constants"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// Log is the logger for predicates.
var Log logr.Logger = log.Log

// EvalGeneric returns true if all predicates match for the given object.
func EvalGeneric(obj runtime.Object, predicates ...predicate.Predicate) bool {
	e := extensionsevent.NewFromObject(obj)

	for _, p := range predicates {
		if !p.Generic(e) {
			return false
		}
	}

	return true
}

type shootNotFailedMapper struct {
	log logr.Logger
	extensionsinject.WithClient
	extensionsinject.WithContext
	extensionsinject.WithCache
}

func (s *shootNotFailedMapper) Map(e event.GenericEvent) bool {
	if e.Meta == nil {
		return false
	}

	// Wait for cache sync because of backing client cache.
	if !s.Cache.WaitForCacheSync(s.Context.Done()) {
		err := errors.New("failed to wait for caches to sync")
		s.log.Error(err, "Could not wait for Cache to sync", "predicate", "ShootNotFailed")
		return false
	}

	cluster, err := controller.GetCluster(s.Context, s.Client, e.Meta.GetNamespace())
	if err != nil {
		s.log.Error(err, "Could not retrieve corresponding cluster")
		return false
	}

	lastOperation := cluster.Shoot.Status.LastOperation
	return lastOperation != nil &&
		lastOperation.State != gardencorev1alpha1.LastOperationStateFailed &&
		cluster.Shoot.Generation == cluster.Shoot.Status.ObservedGeneration
}

// ShootNotFailed is a predicate for failed shoots.
func ShootNotFailed() predicate.Predicate {
	return FromMapper(&shootNotFailedMapper{log: Log.WithName("shoot-not-failed")},
		CreateTrigger, UpdateNewTrigger, DeleteTrigger, GenericTrigger)
}

// GenerationChanged is a predicate for generation changes.
func GenerationChanged() predicate.Predicate {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			return e.MetaOld.GetGeneration() != e.MetaNew.GetGeneration()
		},
	}
}

type or struct {
	predicates []predicate.Predicate
}

func (o *or) orRange(f func(predicate.Predicate) bool) bool {
	for _, p := range o.predicates {
		if f(p) {
			return true
		}
	}
	return false
}

// Create implements Predicate.
func (o *or) Create(event event.CreateEvent) bool {
	return o.orRange(func(p predicate.Predicate) bool { return p.Create(event) })
}

// Delete implements Predicate.
func (o *or) Delete(event event.DeleteEvent) bool {
	return o.orRange(func(p predicate.Predicate) bool { return p.Delete(event) })
}

// Update implements Predicate.
func (o *or) Update(event event.UpdateEvent) bool {
	return o.orRange(func(p predicate.Predicate) bool { return p.Update(event) })
}

// Generic implements Predicate.
func (o *or) Generic(event event.GenericEvent) bool {
	return o.orRange(func(p predicate.Predicate) bool { return p.Generic(event) })
}

// InjectFunc implements Injector.
func (o *or) InjectFunc(f inject.Func) error {
	for _, p := range o.predicates {
		if err := f(p); err != nil {
			return err
		}
	}
	return nil
}

// Or builds a logical OR gate of passed predicates.
func Or(predicates ...predicate.Predicate) predicate.Predicate {
	return &or{predicates}
}

// HasType filters the incoming OperatingSystemConfigs for ones that have the same type
// as the given type.
func HasType(typeName string) predicate.Predicate {
	return FromMapper(MapperFunc(func(e event.GenericEvent) bool {
		acc, err := extensions.Accessor(e.Object)
		if err != nil {
			return false
		}

		return acc.GetExtensionSpec().GetExtensionType() == typeName
	}), CreateTrigger, UpdateNewTrigger, DeleteTrigger, GenericTrigger)
}

// HasName returns a predicate that matches the given name of a resource.
func HasName(name string) predicate.Predicate {
	return FromMapper(MapperFunc(func(e event.GenericEvent) bool {
		return e.Meta.GetName() == name
	}), CreateTrigger, UpdateNewTrigger, DeleteTrigger, GenericTrigger)
}

// HasOperationAnnotation is a predicate for the operation annotation.
func HasOperationAnnotation() predicate.Predicate {
	return FromMapper(MapperFunc(func(e event.GenericEvent) bool {
		return e.Meta.GetAnnotations()[v1alpha1constants.GardenerOperation] == v1alpha1constants.GardenerOperationReconcile
	}), CreateTrigger, UpdateNewTrigger, GenericTrigger)
}

// LastOperationNotSuccessful is a predicate for unsuccessful last operations for creation events.
func LastOperationNotSuccessful() predicate.Predicate {
	operationNotSucceeded := func(obj runtime.Object) bool {
		acc, err := extensions.Accessor(obj)
		if err != nil {
			return false
		}

		lastOp := acc.GetExtensionStatus().GetLastOperation()
		return lastOp == nil ||
			lastOp.GetState() != gardencorev1alpha1.LastOperationStateSucceeded
	}

	return predicate.Funcs{
		CreateFunc: func(event event.CreateEvent) bool {
			return operationNotSucceeded(event.Object)
		},
		UpdateFunc: func(event event.UpdateEvent) bool {
			return operationNotSucceeded(event.ObjectNew)
		},
		GenericFunc: func(event event.GenericEvent) bool {
			return operationNotSucceeded(event.Object)
		},
		DeleteFunc: func(event event.DeleteEvent) bool {
			return operationNotSucceeded(event.Object)
		},
	}
}

// IsDeleting is a predicate for objects having a deletion timestamp.
func IsDeleting() predicate.Predicate {
	return FromMapper(MapperFunc(func(e event.GenericEvent) bool {
		return e.Meta.GetDeletionTimestamp() != nil
	}), CreateTrigger, UpdateNewTrigger, GenericTrigger)
}

// AddTypePredicate returns a new slice which contains a type predicate and the given `predicates`.
func AddTypePredicate(extensionType string, predicates []predicate.Predicate) []predicate.Predicate {
	preds := make([]predicate.Predicate, 0, len(predicates)+1)
	preds = append(preds, HasType(extensionType))
	return append(preds, predicates...)
}

// StatusHasChanged is a predicate for unsuccessful last operations for creation events.
func StatusHasChanged() predicate.Predicate {
	statusHasChanged := func(oldObj runtime.Object, newObj runtime.Object) bool {
		oldAcc, err := extensions.Accessor(oldObj)
		if err != nil {
			return false
		}
		newAcc, err := extensions.Accessor(newObj)
		if err != nil {
			return false
		}

		oldStatus := oldAcc.GetExtensionStatus()
		newStatus := newAcc.GetExtensionStatus()

		// Check conditions
		oldConditions := oldStatus.GetConditions()
		newConditions := newStatus.GetConditions()

		if len(oldConditions) != len(newConditions) {
			return true
		}

		oldConditionsMap := make(map[gardencorev1alpha1.ConditionType]*gardencorev1alpha1.Condition, len(oldConditions))
		for conditionIndex, condition := range oldConditions {
			oldConditionsMap[condition.Type] = &oldConditions[conditionIndex]
		}
		for _, newCondition := range newConditions {
			oldCondition, ok := oldConditionsMap[newCondition.Type]
			if !ok {
				return true
			}
			if oldCondition.Status != newCondition.Status ||
				oldCondition.Reason != newCondition.Reason ||
				oldCondition.Message != newCondition.Message ||
				!oldCondition.LastTransitionTime.Equal(&newCondition.LastTransitionTime) ||
				!oldCondition.LastUpdateTime.Equal(&newCondition.LastUpdateTime) {
				return true
			}

		}

		// Check LastOperation
		oldLastOperation := oldStatus.GetLastOperation()
		newLastOperation := newStatus.GetLastOperation()

		oldLastOperationUpdateTime := oldLastOperation.GetLastUpdateTime()
		newLastOperationUpdateTime := newLastOperation.GetLastUpdateTime()

		if oldLastOperation.GetDescription() != newLastOperation.GetDescription() ||
			oldLastOperation.GetProgress() != newLastOperation.GetProgress() ||
			oldLastOperation.GetState() != newLastOperation.GetState() ||
			oldLastOperation.GetType() != newLastOperation.GetType() ||
			!oldLastOperationUpdateTime.Equal(&newLastOperationUpdateTime) {
			return true
		}

		return oldStatus.GetObservedGeneration() != newStatus.GetObservedGeneration() ||
			oldStatus.GetLastError() != newStatus.GetLastError()
	}

	return predicate.Funcs{
		CreateFunc: func(event event.CreateEvent) bool {
			return true
		},
		UpdateFunc: func(event event.UpdateEvent) bool {
			return statusHasChanged(event.ObjectOld, event.ObjectNew)
		},
		GenericFunc: func(event event.GenericEvent) bool {
			return false
		},
		DeleteFunc: func(event event.DeleteEvent) bool {
			return true
		},
	}
}
