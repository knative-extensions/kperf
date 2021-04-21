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

package eventing

import (
	"github.com/spf13/cobra"
	"knative.dev/kperf/pkg"
)

// domainCmd represents the domain command
func NewEventingCmd(p *pkg.PerfParams) *cobra.Command {
	var eventingCmd = &cobra.Command{
		Use:   "eventing",
		Short: "Knative eventing load test",
		Long: `Knative eventing load test and measurement. For example:

kperf eventing measure --range 1,10, --name perf - to measure 10 Knative eventing wokloads in perf-x namespaces (x between 1 and 10)`,
	}
	eventingCmd.AddCommand(NewEventingPrepareCommand(p))
	eventingCmd.AddCommand(NewEventingMeasureCommand(p))
	eventingCmd.AddCommand(NewEventingCleanCommand(p))
	eventingCmd.AddCommand(NewEventingDriverCommand(p))
	eventingCmd.AddCommand(NewEventingReceiverCommand(p))

	eventingCmd.InitDefaultHelpCmd()
	return eventingCmd
}
