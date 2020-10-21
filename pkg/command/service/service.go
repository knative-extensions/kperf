// Copyright © 2020 The Knative Authors
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

package service

import (
	"github.com/spf13/cobra"
	"knative.dev/kperf/pkg"
)

// domainCmd represents the domain command
func NewServiceCmd(p *pkg.PerfParams) *cobra.Command {
	var serviceCmd = &cobra.Command{
		Use:   "service",
		Short: "Knative service load test",
		Long: `Knative service load test and measurement. For example:

kperf service measurement --range 1,10, --name perf - to measure 10 Knative service named perf-x in perf-x namespace`,
	}
	serviceCmd.AddCommand(NewServiceMeasureCommand(p))
	serviceCmd.AddCommand(NewServiceGenerateCommand(p))
	serviceCmd.AddCommand(NewServiceCleanCommand(p))

	serviceCmd.InitDefaultHelpCmd()
	return serviceCmd
}
