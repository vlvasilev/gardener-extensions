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

package worker_test

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/gardener/gardener-extensions/pkg/controller"
	extensionscontroller "github.com/gardener/gardener-extensions/pkg/controller"
	"github.com/gardener/gardener-extensions/pkg/controller/worker"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	extensionsclient "github.com/gardener/gardener/pkg/client/extensions/clientset/versioned/scheme"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
)

var _ = Describe("Worker Reconcile", func() {
	type fields struct {
		logger   logr.Logger
		actuator worker.Actuator
		ctx      context.Context
		client   client.Client
	}
	type args struct {
		request reconcile.Request
	}
	type test struct {
		fields  fields
		args    args
		want    reconcile.Result
		wantErr bool
	}

	//Ummutable throuth the function calls
	arguments := args{
		request: reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      "workerTestReconcile",
				Namespace: "test",
			},
		},
	}

	var logger logr.Logger

	BeforeEach(func() {
		logger = log.Log.WithName("Reconcile-Test-Controller")
	})

	DescribeTable("Reconcile function", func(t *test) {
		reconciler := worker.NewReconciler(nil, t.fields.actuator)
		expectInject(inject.ClientInto(t.fields.client, reconciler))
		expectInject(inject.InjectorInto(inject.Func(func(i interface{}) error {
			expectInject(inject.ClientInto(t.fields.client, i))
			expectInject(inject.StopChannelInto(make(chan struct{}), i))
			return nil
		}), reconciler))

		got, err := reconciler.Reconcile(t.args.request)
		Expect(err != nil).To(Equal(t.wantErr))
		Expect(reflect.DeepEqual(got, t.want)).To(BeTrue())
	},
		Entry("test reconcile", &test{
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
		}),
		Entry("test after successful migrate", &test{
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
		}),
		Entry("test migrate when operrationAnnotation Migrate occurs", &test{
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
		}),
		Entry("test error during migrate when operrationAnnotation Migrate occurs", &test{
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
		}),
		Entry("test Migrate after unssuccesful Migrate", &test{
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
		}),
		Entry("test error during Migrate after unssuccesful Migrate", &test{
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
		}),
		Entry("test Delete Worker", &test{
			fields: fields{
				logger:   logger,
				actuator: newFakeActuator(false, true, false, false),
				ctx:      context.TODO(),
				client: fake.NewFakeClientWithScheme(
					extensionsclient.Scheme,
					addFinalizerToWorker(addDeletionTimestampToWorker(getWorker()), worker.FinalizerName),
					getCluster()),
			},
			args:    arguments,
			want:    reconcile.Result{},
			wantErr: false,
		}),
		Entry("test error when Delete Worker", &test{
			fields: fields{
				logger:   logger,
				actuator: newFakeActuator(true, false, true, true),
				ctx:      context.TODO(),
				client: fake.NewFakeClientWithScheme(
					extensionsclient.Scheme,
					addFinalizerToWorker(addDeletionTimestampToWorker(getWorker()), worker.FinalizerName),
					getCluster()),
			},
			args:    arguments,
			want:    reconcile.Result{},
			wantErr: true,
		}),
		Entry("test restore when operrationAnnotation Restore occurs", &test{
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
			want:    reconcile.Result{Requeue: true},
			wantErr: false,
		}),
		Entry("test error restore when operrationAnnotation Restore occurs", &test{
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
		}),
		Entry("test reconcile after failed reconcilation", &test{
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
		}),
		Entry("test reconcile after successful restoration reconcilation", &test{
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
		}),
		Entry("test error while reconciliation after failed reconcilation", &test{
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
		}),
		Entry("test error while reconciliation after successful restoration reconcilation", &test{
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
		}),
	)
})

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

type fakeActuator struct {
	reconcile bool
	delete    bool
	restore   bool
	migrate   bool
}

func newFakeActuator(reconcile, delete, restore, migrate bool) worker.Actuator {
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

func expectInject(ok bool, err error) {
	Expect(err).NotTo(HaveOccurred())
	Expect(ok).To(BeTrue(), "no injection happened")
}
