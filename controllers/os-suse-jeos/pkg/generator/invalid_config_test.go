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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"os"
)

var _ = Describe("JeOS Cloud-init Generator Test", func() {

	It("should fail creating generator without OS_CONFIG_FORMAT variable", func() {
		os.Unsetenv(OsConfigFormat)
		os.Setenv(BootCommand, "boot-command")
		_, err := NewCloudInitGenerator()

		Expect(err).To(HaveOccurred())
	})

	It("should fail creating generator with empty OS_CONFIG_FORMAT variable", func() {
		os.Setenv(OsConfigFormat, "")
		os.Setenv(BootCommand, "boot-command")
		_, err := NewCloudInitGenerator()

		Expect(err).To(HaveOccurred())
	})

	It("should fail creating generator with empty BOOT_COMMAND variable", func() {
		os.Setenv(OsConfigFormat, "")
		os.Setenv(BootCommand, "")
		_, err := NewCloudInitGenerator()

		Expect(err).To(HaveOccurred())
	})

	It("should fail creating generator without BOOT_COMMAND variable", func() {
		os.Setenv(OsConfigFormat, "os-format")
		os.Unsetenv(BootCommand)
		_, err := NewCloudInitGenerator()

		Expect(err).To(HaveOccurred())
	})

	It("should fail creating generator with invalid OS_CONFIG_FORMAT", func() {
		os.Setenv(OsConfigFormat, "invalid-format")
		os.Setenv(BootCommand, "some command")
		_, err := NewCloudInitGenerator()

		Expect(err).To(HaveOccurred())
	})
})