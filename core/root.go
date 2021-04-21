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
	"fmt"
	"os"

	"knative.dev/kperf/pkg/command/eventing"
	"knative.dev/kperf/pkg/command/service"
	"knative.dev/kperf/pkg/command/version"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"knative.dev/kperf/pkg"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands

func NewPerfCommand(params ...pkg.PerfParams) *cobra.Command {
	p := &pkg.PerfParams{}
	p.Initialize()

	rootCmd := &cobra.Command{
		Use:   "kperf",
		Short: "A CLI to help with Knative performance test",
		Long:  `A CLI to help with Knative performance test.`,
	}
	cobra.OnInitialize(initConfig)
	rootCmd.AddCommand(eventing.NewEventingCmd(p))
	rootCmd.AddCommand(service.NewServiceCmd(p))
	rootCmd.AddCommand(version.NewVersionCommand())
	rootCmd.InitDefaultHelpCmd()
	return rootCmd
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		viper.AddConfigPath(home)
		viper.SetConfigName(".kperf")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
