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

package app

import (
	"context"
	"fmt"
	"os"

	awscmd "github.com/gardener/gardener-extensions/controllers/provider-aws/pkg/cmd"
	"github.com/gardener/gardener-extensions/controllers/provider-aws/pkg/infrastructure"
	"github.com/gardener/gardener-extensions/controllers/provider-aws/pkg/register"
	"github.com/gardener/gardener-extensions/controllers/provider-aws/pkg/worker"
	"github.com/gardener/gardener-extensions/pkg/controller"
	controllercmd "github.com/gardener/gardener-extensions/pkg/controller/cmd"

	"github.com/spf13/cobra"

	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// Name is the name of the AWS provider controller.
const Name = "provider-aws-controller"

// NewControllerCommand creates a new command for running an AWS provider controller.
func NewControllerCommand(ctx context.Context) *cobra.Command {
	var (
		restOpts = &controllercmd.RESTOptions{}
		mgrOpts  = &controllercmd.ManagerOptions{
			LeaderElection:          true,
			LeaderElectionNamespace: os.Getenv("LEADER_ELECTION_NAMESPACE"),
		}
		infraCtrlOpts = &controllercmd.ControllerOptions{
			MaxConcurrentReconciles: 5,
		}
		workerCtrlOpts = &controllercmd.ControllerOptions{
			MaxConcurrentReconciles: 5,
		}
		configFileOpts = &awscmd.ConfigOptions{}

		aggOption = controllercmd.NewOptionAggregator(
			restOpts,
			mgrOpts,
			configFileOpts,
			controllercmd.PrefixOption("infrastructure-", infraCtrlOpts),
			controllercmd.PrefixOption("worker-", workerCtrlOpts),
		)
	)

	cmd := &cobra.Command{
		Use: fmt.Sprintf("%s-manager", Name),

		Run: func(cmd *cobra.Command, args []string) {
			if err := aggOption.Complete(); err != nil {
				controllercmd.LogErrAndExit(err, "Error completing options")
			}

			mgr, err := manager.New(restOpts.Completed().Config, mgrOpts.Completed().Options())
			if err != nil {
				controllercmd.LogErrAndExit(err, "Could not instantiate manager")
			}

			if err := controller.AddToScheme(mgr.GetScheme()); err != nil {
				controllercmd.LogErrAndExit(err, "Could not update manager scheme")
			}

			infraCtrlOpts.Completed().Apply(&infrastructure.Options)
			workerCtrlOpts.Completed().Apply(&worker.Options)
			configFileOpts.Completed().ApplyMachineImages(&worker.MachineImages)

			if err := register.AddToManager(mgr); err != nil {
				controllercmd.LogErrAndExit(err, "Could not add controllers to manager")
			}

			if err := mgr.Start(ctx.Done()); err != nil {
				controllercmd.LogErrAndExit(err, "Error running manager")
			}
		},
	}

	aggOption.AddFlags(cmd.Flags())

	return cmd
}
