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

package worker

import (
	"github.com/go-logr/logr"
)

// ActuatorFactory is an abstract factory used for creating Actuators.
type ActuatorFactory func(*ActuatorArgs) (Actuator, error)

// ActuatorArgs are arguments given to the instantiation of an Actuator.
type ActuatorArgs struct {
	Log logr.Logger
}

// Actuator acts upon Infrastructure resources.
type Actuator interface {
	// DeployMachineControllerManager must deploy the machine-controller-manager into the namespace.
	DeployMachineControllerManager() error
	// GetMachineClassInfo returns information about the MachineClass.
	GetMachineClassInfo() (string, string, string)
}
