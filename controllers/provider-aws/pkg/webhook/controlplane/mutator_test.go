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

package controlplane

import (
	"context"
	"testing"

	"github.com/gardener/gardener-extensions/controllers/provider-aws/pkg/aws"
	"github.com/gardener/gardener-extensions/pkg/webhook/controlplane"
	"github.com/gardener/gardener-extensions/pkg/webhook/controlplane/test"

	"github.com/gardener/gardener/pkg/operation/common"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestController(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AWS Controlplane Webhook Suite")
}

var _ = Describe("Mutator", func() {

	Describe("#Mutate", func() {
		It("should add missing elements to kube-apiserver deployment", func() {
			var (
				dep = &appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{Name: common.KubeAPIServerDeploymentName},
					Spec: appsv1.DeploymentSpec{
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{
									{
										Name: "kube-apiserver",
									},
								},
							},
						},
					},
				}
			)

			mutator := newMutator(logger)
			err := mutator.Mutate(context.TODO(), dep)
			Expect(err).To(Not(HaveOccurred()))
			checkKubeAPIServerDeployment(dep)
		})

		It("should modify existing elements of kube-apiserver deployment", func() {
			var (
				dep = &appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{Name: common.KubeAPIServerDeploymentName},
					Spec: appsv1.DeploymentSpec{
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{
									{
										Name: "kube-apiserver",
										Command: []string{
											"--cloud-provider=?",
											"--cloud-config=?",
											"--enable-admission-plugins=Priority,NamespaceLifecycle",
											"--disable-admission-plugins=PersistentVolumeLabel",
										},
										Env: []corev1.EnvVar{
											{Name: "AWS_ACCESS_KEY_ID", Value: "?"},
											{Name: "AWS_SECRET_ACCESS_KEY", Value: "?"},
										},
										VolumeMounts: []corev1.VolumeMount{
											{Name: aws.CloudProviderConfigName, MountPath: "?"},
											// TODO Use constant from github.com/gardener/gardener/pkg/apis/core/v1alpha1 when available
											// See https://github.com/gardener/gardener/pull/930
											{Name: common.CloudProviderSecretName, MountPath: "?"},
										},
									},
								},
								Volumes: []corev1.Volume{
									{Name: aws.CloudProviderConfigName},
									{Name: common.CloudProviderSecretName},
								},
							},
						},
					},
				}
			)

			mutator := newMutator(logger)
			err := mutator.Mutate(context.TODO(), dep)
			Expect(err).To(Not(HaveOccurred()))
			checkKubeAPIServerDeployment(dep)
		})

		It("should add missing elements to kube-controller-manager deployment", func() {
			var (
				dep = &appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{Name: common.KubeControllerManagerDeploymentName},
					Spec: appsv1.DeploymentSpec{
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{
									{
										Name: "kube-controller-manager",
									},
								},
							},
						},
					},
				}
			)

			mutator := newMutator(logger)
			err := mutator.Mutate(context.TODO(), dep)
			Expect(err).To(Not(HaveOccurred()))
			checkKubeControllerManagerDeployment(dep)
		})

		It("should modify existing elements of kube-controller-manager deployment", func() {
			var (
				dep = &appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{Name: common.KubeControllerManagerDeploymentName},
					Spec: appsv1.DeploymentSpec{
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{
									{
										Name: "kube-controller-manager",
										Command: []string{
											"--cloud-provider=?",
											"--cloud-config=?",
											"--external-cloud-volume-plugin=?",
										},
										Env: []corev1.EnvVar{
											{Name: "AWS_ACCESS_KEY_ID", Value: "?"},
											{Name: "AWS_SECRET_ACCESS_KEY", Value: "?"},
										},
										VolumeMounts: []corev1.VolumeMount{
											{Name: aws.CloudProviderConfigName, MountPath: "?"},
											{Name: common.CloudProviderSecretName, MountPath: "?"},
										},
									},
								},
								Volumes: []corev1.Volume{
									{Name: aws.CloudProviderConfigName},
									{Name: common.CloudProviderSecretName},
								},
							},
						},
					},
				}
			)

			mutator := newMutator(logger)
			err := mutator.Mutate(context.TODO(), dep)
			Expect(err).To(Not(HaveOccurred()))
			checkKubeControllerManagerDeployment(dep)
		})
	})
})

func checkKubeAPIServerDeployment(dep *appsv1.Deployment) {
	// Check that the kube-apiserver container still exists and contains all needed command line args,
	// env vars, and volume mounts
	c := controlplane.ContainerWithName(dep.Spec.Template.Spec.Containers, "kube-apiserver")
	Expect(c).To(Not(BeNil()))
	Expect(c.Command).To(ContainElement("--cloud-provider=aws"))
	Expect(c.Command).To(ContainElement("--cloud-config=/etc/kubernetes/cloudprovider/cloudprovider.conf"))
	Expect(c.Command).To(test.ContainElementWithPrefixContaining("--enable-admission-plugins=", "PersistentVolumeLabel", ","))
	Expect(c.Command).To(Not(test.ContainElementWithPrefixContaining("--disable-admission-plugins=", "PersistentVolumeLabel", ",")))
	Expect(c.Env).To(ContainElement(accessKeyIDEnvVar))
	Expect(c.Env).To(ContainElement(secretAccessKeyEnvVar))
	Expect(c.VolumeMounts).To(ContainElement(cloudProviderConfigVolumeMount))
	Expect(c.VolumeMounts).To(ContainElement(cloudProviderSecretVolumeMount))

	// Check that the Pod spec contains all needed volumes
	Expect(dep.Spec.Template.Spec.Volumes).To(ContainElement(cloudProviderConfigVolume))
	Expect(dep.Spec.Template.Spec.Volumes).To(ContainElement(cloudProviderSecretVolume))
}

func checkKubeControllerManagerDeployment(dep *appsv1.Deployment) {
	// Check that the kube-controller-manager container still exists and contains all needed command line args,
	// env vars, and volume mounts
	c := controlplane.ContainerWithName(dep.Spec.Template.Spec.Containers, "kube-controller-manager")
	Expect(c).To(Not(BeNil()))
	Expect(c.Command).To(ContainElement("--cloud-provider=external"))
	Expect(c.Command).To(ContainElement("--cloud-config=/etc/kubernetes/cloudprovider/cloudprovider.conf"))
	Expect(c.Command).To(ContainElement("--external-cloud-volume-plugin=aws"))
	Expect(c.Env).To(ContainElement(accessKeyIDEnvVar))
	Expect(c.Env).To(ContainElement(secretAccessKeyEnvVar))
	Expect(c.VolumeMounts).To(ContainElement(cloudProviderConfigVolumeMount))
	Expect(c.VolumeMounts).To(ContainElement(cloudProviderSecretVolumeMount))

	// Check that the Pod spec contains all needed volumes
	Expect(dep.Spec.Template.Spec.Volumes).To(ContainElement(cloudProviderConfigVolume))
	Expect(dep.Spec.Template.Spec.Volumes).To(ContainElement(cloudProviderSecretVolume))
}
