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

package generator

import (
	"io/ioutil"

	"fmt"

	actuator "github.com/gardener/gardener-extensions/pkg/controller/operatingsystemconfig/oscommon/actuator"
	"github.com/gardener/gardener-extensions/pkg/controller/operatingsystemconfig/oscommon/cloudinit"
	commonosgenerator "github.com/gardener/gardener-extensions/pkg/controller/operatingsystemconfig/oscommon/generator"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	yaml "gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Ubuntu OS Generator Test", func() {
	//var box = packr.NewBox("./testfiles")
	generator, err := NewCloudInitGenerator()

	It("should not fail creating generator", func() {
		Expect(err).NotTo(HaveOccurred())
	})

	osc := &extensionsv1alpha1.OperatingSystemConfig{}

	b, err := ioutil.ReadFile("../../example/operatingsystemconfig.yaml")
	It("should not fail reading ../../example/operatingsystemconfig.yaml", func() {
		Expect(err).NotTo(HaveOccurred())
	})

	err = yaml.Unmarshal(b, osc)
	It("should not fail unmarshal ../../example/operatingsystemconfig.yaml", func() {
		Expect(err).NotTo(HaveOccurred())
	})
	m := new(map[string]interface{})

	err = yaml.Unmarshal(b, m)
	It("should not fail unmarshal ../../example/operatingsystemconfig.yaml", func() {
		Expect(err).NotTo(HaveOccurred())
	})
	fmt.Println(osc)
	fmt.Println(m)

	cloudConfig, cmd, unitsNames, err := reconcile(osc, generator)
	It("should not fail reconsilation", func() {
		Expect(err).NotTo(HaveOccurred())
	})

	fmt.Println(cloudConfig)
	fmt.Println(cmd)
	fmt.Println(unitsNames)
	//Describe("Conformance Tests", test.DescribeTest(generator, box))
})

// Reconcile reconciles the update of a OperatingSystemConfig regenerating the os-specific format
func reconcile(config *extensionsv1alpha1.OperatingSystemConfig, generator commonosgenerator.Generator) ([]byte, *string, []string, error) {

	cloudConfig, cmd, err := cloudConfigFromOperatingSystemConfig(config, generator)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("could not generate cloud config: %v", err)
	}

	return []byte(cloudConfig), cmd, actuator.OperatingSystemConfigUnitNames(config), nil
}

func cloudConfigFromOperatingSystemConfig(config *extensionsv1alpha1.OperatingSystemConfig, generator commonosgenerator.Generator) ([]byte, *string, error) {
	files := make([]*commonosgenerator.File, 0, len(config.Spec.Files))
	for _, file := range config.Spec.Files {
		if file.Content.Inline == nil {
			fmt.Println("!!!!!!!!!!!!!!!!!file.Content.Inline!!!!!!!!!!!!!!!!!!")
		}
		fmt.Printf("!!!!!!!!!!!!!!!!!file.Name %s!!!!!!!!!!!!!!!!!!", file.Path)
		data, err := dataForFileContent(config.Namespace, &file.Content)
		if err != nil {
			return nil, nil, err
		}

		files = append(files, &commonosgenerator.File{Path: file.Path, Content: data, Permissions: file.Permissions})
	}

	units := make([]*commonosgenerator.Unit, 0, len(config.Spec.Units))
	for _, unit := range config.Spec.Units {
		var content []byte
		if unit.Content != nil {
			content = []byte(*unit.Content)
		}

		dropIns := make([]*commonosgenerator.DropIn, 0, len(unit.DropIns))
		for _, dropIn := range unit.DropIns {
			dropIns = append(dropIns, &commonosgenerator.DropIn{Name: dropIn.Name, Content: []byte(dropIn.Content)})
		}
		units = append(units, &commonosgenerator.Unit{Name: unit.Name, Content: content, DropIns: dropIns})
	}

	return generator.Generate(&commonosgenerator.OperatingSystemConfig{
		Bootstrap: config.Spec.Purpose == extensionsv1alpha1.OperatingSystemConfigPurposeProvision,
		Files:     files,
		Units:     units,
		Path:      config.Spec.ReloadConfigFilePath,
	})
}

// DataForFileContent returns the content for a FileContent, retrieving from a Secret if necessary.
func dataForFileContent(namespace string, content *extensionsv1alpha1.FileContent) ([]byte, error) {
	if inline := content.Inline; inline != nil {
		if len(inline.Encoding) == 0 {
			return []byte(inline.Data), nil
		}
		return cloudinit.Decode(inline.Encoding, []byte(inline.Data))
	}
	fmt.Println("Here")
	if content.SecretRef == nil {
		fmt.Println("content.SecretRef is nil")
		return []byte{}, nil
	}
	return []byte(fmt.Sprintf("find the data in the secret %s under key %s. The namespace is %s..", content.SecretRef.Name, content.SecretRef.DataKey, namespace)), nil
}

func fromMapToOperatingSystemConfig(oscMap map[string]interface{}) *extensionsv1alpha1.OperatingSystemConfig {
	if oscMap == nil {
		return nil
	}

	osc := &extensionsv1alpha1.OperatingSystemConfig{}

	metaData, ok := oscMap["metadata"].(map[string]string)
	if ok {
		osc.Name = metaData["name"]
		osc.Namespace = metaData["namespace"]
	}

}

type OperatingSystemConfig struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OperatingSystemConfigSpec   `json:"spec"`
	Status OperatingSystemConfigStatus `json:"status"`
}
