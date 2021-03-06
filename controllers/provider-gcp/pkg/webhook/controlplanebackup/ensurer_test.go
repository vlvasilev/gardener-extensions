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

package controlplanebackup

import (
	"context"
	"path"
	"testing"

	"github.com/gardener/gardener-extensions/controllers/provider-gcp/pkg/apis/config"
	"github.com/gardener/gardener-extensions/controllers/provider-gcp/pkg/gcp"
	extensionscontroller "github.com/gardener/gardener-extensions/pkg/controller"
	"github.com/gardener/gardener-extensions/pkg/util"
	"github.com/gardener/gardener-extensions/pkg/webhook/controlplane"

	gardenv1beta1 "github.com/gardener/gardener/pkg/apis/garden/v1beta1"
	"github.com/gardener/gardener/pkg/operation/common"
	"github.com/gardener/gardener/pkg/utils/imagevector"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestController(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "GCP Controlplane Backup Webhook Suite")
}

var _ = Describe("Ensurer", func() {
	Describe("#EnsureETCDStatefulSet", func() {
		var (
			etcdBackup = &config.ETCDBackup{
				Schedule: util.StringPtr("0 */24 * * *"),
			}

			imageVector = imagevector.ImageVector{
				{
					Name:       gcp.ETCDBackupRestoreImageName,
					Repository: "test-repository",
					Tag:        "test-tag",
				},
			}

			cluster = &extensionscontroller.Cluster{
				Shoot: &gardenv1beta1.Shoot{
					Spec: gardenv1beta1.ShootSpec{
						Kubernetes: gardenv1beta1.Kubernetes{
							Version: "1.13.4",
						},
					},
				},
			}
		)

		It("should add or modify elements to etcd-main statefulset", func() {
			var (
				ss = &appsv1.StatefulSet{
					ObjectMeta: metav1.ObjectMeta{Name: common.EtcdMainStatefulSetName},
				}
			)

			// Create ensurer
			ensurer := NewEnsurer(etcdBackup, imageVector, logger)

			// Call EnsureETCDStatefulSet method and check the result
			err := ensurer.EnsureETCDStatefulSet(context.TODO(), ss, cluster)
			Expect(err).To(Not(HaveOccurred()))
			checkETCDMainStatefulSet(ss)
		})

		It("should modify existing elements of etcd-main statefulset", func() {
			var (
				ss = &appsv1.StatefulSet{
					ObjectMeta: metav1.ObjectMeta{Name: common.EtcdMainStatefulSetName},
					Spec: appsv1.StatefulSetSpec{
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{
									{
										Name: "backup-restore",
									},
								},
							},
						},
					},
				}
			)

			// Create ensurer
			ensurer := NewEnsurer(etcdBackup, imageVector, logger)

			// Call EnsureETCDStatefulSet method and check the result
			err := ensurer.EnsureETCDStatefulSet(context.TODO(), ss, cluster)
			Expect(err).To(Not(HaveOccurred()))
			checkETCDMainStatefulSet(ss)
		})

		It("should add or modify elements to etcd-events statefulset", func() {
			var (
				ss = &appsv1.StatefulSet{
					ObjectMeta: metav1.ObjectMeta{Name: common.EtcdEventsStatefulSetName},
				}
			)

			// Create ensurer
			ensurer := NewEnsurer(etcdBackup, imageVector, logger)

			// Call EnsureETCDStatefulSet method and check the result
			err := ensurer.EnsureETCDStatefulSet(context.TODO(), ss, cluster)
			Expect(err).To(Not(HaveOccurred()))
			checkETCDEventsStatefulSet(ss)
		})

		It("should modify existing elements of etcd-events statefulset", func() {
			var (
				ss = &appsv1.StatefulSet{
					ObjectMeta: metav1.ObjectMeta{Name: common.EtcdEventsStatefulSetName},
					Spec: appsv1.StatefulSetSpec{
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{
									{
										Name: "backup-restore",
									},
								},
							},
						},
					},
				}
			)

			// Create ensurer
			ensurer := NewEnsurer(etcdBackup, imageVector, logger)

			// Call EnsureETCDStatefulSet method and check the result
			err := ensurer.EnsureETCDStatefulSet(context.TODO(), ss, cluster)
			Expect(err).To(Not(HaveOccurred()))
			checkETCDEventsStatefulSet(ss)
		})
	})
})

func checkETCDMainStatefulSet(ss *appsv1.StatefulSet) {
	var (
		env = []corev1.EnvVar{
			{
				Name: "STORAGE_CONTAINER",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						Key:                  gcp.BucketName,
						LocalObjectReference: corev1.LocalObjectReference{Name: gcp.BackupSecretName},
					},
				},
			},
			{
				Name:  "GOOGLE_APPLICATION_CREDENTIALS",
				Value: path.Join("/root/.gcp/", gcp.ServiceAccountJSONField),
			},
		}
		volumeMounts = []corev1.VolumeMount{
			{
				Name:      gcp.BackupSecretName,
				MountPath: "/root/.gcp/",
			},
		}
		etcdBackupSecretVolume = corev1.Volume{
			Name: gcp.BackupSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: gcp.BackupSecretName,
				},
			},
		}
	)

	c := controlplane.ContainerWithName(ss.Spec.Template.Spec.Containers, "backup-restore")
	Expect(c).To(Equal(controlplane.GetBackupRestoreContainer(common.EtcdMainStatefulSetName, "0 */24 * * *", gcp.StorageProviderName,
		"test-repository:test-tag", nil, env, volumeMounts)))
	Expect(ss.Spec.Template.Spec.Volumes).To(ContainElement(etcdBackupSecretVolume))

}

func checkETCDEventsStatefulSet(ss *appsv1.StatefulSet) {
	c := controlplane.ContainerWithName(ss.Spec.Template.Spec.Containers, "backup-restore")
	Expect(c).To(Equal(controlplane.GetBackupRestoreContainer(common.EtcdEventsStatefulSetName, "0 */24 * * *", "",
		"test-repository:test-tag", nil, nil, nil)))
	Expect(ss.Spec.Template.Spec.Volumes).To(BeEmpty())
}
