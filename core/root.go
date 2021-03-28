// Copyright 2020 The Knative Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package core

import (
	"github.com/spf13/cobra"
	"knative.dev/kperf/pkg"
	"knative.dev/kperf/pkg/command/service"
	"knative.dev/kperf/pkg/command/version"
	"knative.dev/kperf/pkg/config"
	"log"
)

//var cfgFile string

// rootCmd represents the base command when called without any subcommands

func NewPerfCommand(params ...pkg.PerfParams) *cobra.Command {
	p := &pkg.PerfParams{}
	p.Initialize()

	rootCmd := &cobra.Command{
		Use:   "kperf",
		Short: "A CLI to help with Knative performance test",
		Long:  `A CLI to help with Knative performance test, like generate workload, measure and visualize result.`,
	}

	rootCmd.AddCommand(service.NewServiceCmd(p))
	rootCmd.AddCommand(version.NewVersionCommand())

	cobra.OnInitialize(initConfig)

	config.AddBootstrapFlags(rootCmd.PersistentFlags())

	rootCmd.InitDefaultHelpCmd()
	return rootCmd
}

func initConfig() {
	err := config.BootstrapConfig()
	if err != nil {
		log.Println("failed to bootstrap config:", err)
	}
}
