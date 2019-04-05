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

package v1alpha1

import (
	gardencorev1alpha1 "github.com/gardener/gardener/pkg/apis/core/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// InfrastructureConfig infrastructure configuration resource
type InfrastructureConfig struct {
	metav1.TypeMeta `json:",inline"`

	// Networks is the network configuration (VPC, subnets, etc.)
	Networks NetworkConfig `json:"networks"`
}

// NetworkConfig holds information about the Kubernetes and infrastructure networks.
type NetworkConfig struct {
	// VPC indicates information about the VPC to create / use.
	VPC VPC `json:"vpc"`

	// Zones indicate the zones in which subnets shall be created.
	// +optional
	Zones []Zone `json:"zones,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// InfrastructureStatus contains information about created infrastructure resources.
type InfrastructureStatus struct {
	metav1.TypeMeta `json:",inline"`

	// Networks is the status of the networks of the infrastructure.
	// +optional
	Networks *NetworkStatus `json:"networks,omitempty"`

	// ServiceAccountEmail is the email address of the service account.
	// +optional
	ServiceAccountEmail *string `json:"serviceAccountEmail,omitempty"`
}

// NetworkStatus is the current status of the infrastructure networks.
type NetworkStatus struct {
	// VPC states the name of the infrastructure VPC.
	VPC VPC `json:"vpc"`

	// Subnets are the subnets that have been created.
	// +optional
	Subnets []Subnet `json:"subnets,omitempty"`
}

type Zone struct {
	// Name is the name of the zone.
	Name string `json:"name"`
	// Worker is the worker CIDR.
	Worker gardencorev1alpha1.CIDR `json:"worker"`
}

// SubnetPurpose is a purpose of a subnet.
type SubnetPurpose string

const (
	// PurposeNodes is a SubnetPurpose for nodes.
	PurposeNodes SubnetPurpose = "nodes"
	// PurposeInternal is a SubnetPurpose for internal use.
	PurposeInternal SubnetPurpose = "internal"
)

// Subnet is a subnet that was created.
type Subnet struct {
	// Name is the name of the subnet.
	Name string `json:"name"`
	// Purpose is the purpose for which the subnet was created.
	Purpose SubnetPurpose `json:"purpose"`
}

// VPC contains information about the VPC and some related resources.
type VPC struct {
	// ID is the VPC ID.
	// +optional
	ID *string `json:"id,omitempty"`

	// CIDR is the CIDR of the VPC.
	// +optional
	CIDR *gardencorev1alpha1.CIDR `json:"cidr,omitempty"`
}
