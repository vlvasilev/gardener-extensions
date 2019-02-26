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

package cmd

import (
	"fmt"
	"io/ioutil"

	"github.com/gardener/gardener-extensions/controllers/provider-aws/pkg/apis/config"
	configv1alpha1 "github.com/gardener/gardener-extensions/controllers/provider-aws/pkg/apis/config/v1alpha1"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	"github.com/spf13/pflag"
)

// ConfigOptions are command line options that can be set for config.ControllerConfiguration.
type ConfigOptions struct {
	// Kubeconfig is the path to a kubeconfig.
	ConfigFilePath string

	scheme *runtime.Scheme
	codecs serializer.CodecFactory
	config *Config
}

// Config is a completed controller configuration.
type Config struct {
	// Config is the controller configuration.
	Config *config.ControllerConfiguration
}

func (c *ConfigOptions) buildConfig() (*config.ControllerConfiguration, error) {
	if len(c.ConfigFilePath) == 0 {
		return nil, fmt.Errorf("config file path not set")
	}

	c.scheme = runtime.NewScheme()
	c.codecs = serializer.NewCodecFactory(c.scheme)

	if err := config.AddToScheme(c.scheme); err != nil {
		return nil, err
	}
	if err := configv1alpha1.AddToScheme(c.scheme); err != nil {
		return nil, err
	}

	cfg, err := c.loadConfigFromFile(c.ConfigFilePath)
	if err != nil {
		return nil, err
	}

	cfg, err = c.applyDefaults(cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *ConfigOptions) loadConfigFromFile(path string) (*config.ControllerConfiguration, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return c.decodeConfig(data)
}

// decodeConfig decodes data as a ControllerConfiguration object.
func (c *ConfigOptions) decodeConfig(data []byte) (*config.ControllerConfiguration, error) {
	configObj, gvk, err := c.codecs.UniversalDecoder().Decode(data, nil, nil)
	if err != nil {
		return nil, err
	}
	config, ok := configObj.(*config.ControllerConfiguration)
	if !ok {
		return nil, fmt.Errorf("got unexpected config type: %v", gvk)
	}
	return config, nil
}

func (c *ConfigOptions) applyDefaults(in *config.ControllerConfiguration) (*config.ControllerConfiguration, error) {
	external, err := c.scheme.ConvertToVersion(in, configv1alpha1.SchemeGroupVersion)
	if err != nil {
		return nil, err
	}
	c.scheme.Default(external)

	internal, err := c.scheme.ConvertToVersion(external, config.SchemeGroupVersion)
	if err != nil {
		return nil, err
	}
	out := internal.(*config.ControllerConfiguration)

	return out, nil
}

// Complete implements RESTCompleter.Complete.
func (c *ConfigOptions) Complete() error {
	config, err := c.buildConfig()
	if err != nil {
		return err
	}

	c.config = &Config{config}
	return nil
}

// Completed returns the completed Config. Only call this if `Complete` was successful.
func (c *ConfigOptions) Completed() *Config {
	return c.config
}

// AddFlags implements Flagger.AddFlags.
func (c *ConfigOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.ConfigFilePath, "config-file", "", "path to the controller manager configuration file")
}

// Apply sets the values of this Config in the given config.ControllerConfiguration.
func (c *Config) Apply(cfg *config.ControllerConfiguration) {
	*cfg = *c.Config
}

// ApplyMachineImages sets the given machine images to those of this Config.
func (c *Config) ApplyMachineImages(machineImages *[]config.MachineImage) {
	*machineImages = c.Config.MachineImages
}

// Options initializes empty config.ControllerConfiguration, applies the set values and returns it.
func (c *Config) Options() config.ControllerConfiguration {
	var cfg config.ControllerConfiguration
	c.Apply(&cfg)
	return cfg
}
