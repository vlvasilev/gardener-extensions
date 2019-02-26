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
	"github.com/gardener/gardener-extensions/controllers/provider-aws/pkg/apis/config"
	"github.com/gardener/gardener-extensions/pkg/controller/worker"

	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"

	"github.com/go-logr/logr"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

type actuator struct {
	logger logr.Logger

	restConfig    *rest.Config
	machineImages []config.MachineImage

	client     client.Client
	kubernetes kubernetes.Interface
	scheme     *runtime.Scheme
	decoder    runtime.Decoder
	encoder    runtime.Encoder
}

// NewActuator creates a new Actuator that updates the status of the handled WorkerPoolConfigs.
func NewActuator(machineImages []config.MachineImage) worker.Actuator {
	return &actuator{
		logger:        log.Log.WithName("worker-actuator"),
		machineImages: machineImages,
	}
}

func (c *actuator) InjectScheme(scheme *runtime.Scheme) error {
	c.scheme = scheme
	c.decoder = serializer.NewCodecFactory(c.scheme).UniversalDecoder()
	c.encoder = serializer.NewCodecFactory(c.scheme).EncoderForVersion(c.encoder, extensionsv1alpha1.SchemeGroupVersion)
	return nil
}

func (c *actuator) InjectConfig(restConfig *rest.Config) error {
	c.restConfig = restConfig
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	c.kubernetes = clientset
	return nil
}

func (c *actuator) InjectClient(client client.Client) error {
	c.client = client
	return nil
}

func (c *actuator) DeployMachineControllerManager() error {
	return nil
}

func (c *actuator) GetMachineClassInfo() (string, string, string) {
	return "1", "2", "3"
}
