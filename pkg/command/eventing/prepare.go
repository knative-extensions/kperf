// Copyright 202 The Knative Authors
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
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"knative.dev/kperf/pkg"
	"knative.dev/kperf/pkg/command/eventing/scenario"
)

func NewEventingPrepareCommand(p *pkg.PerfParams) *cobra.Command {
	//generateArgs := generateArgs{}

	var prepareCmd = &cobra.Command{
		Use:   "prepare SCENARIO|COMMAND [options]",
		Short: "Prepare the cluster before running performance tests",
	}

	prepareCmd.AddCommand(NewEventingPrepareListCommand(p))
	for _, ctor := range scenario.GetCommandCtors() {
		prepareCmd.AddCommand(ctor(p))
	}

	// TODO: this will be added later
	// prepareCmd.PersistentFlags().StringVarP(&generateArgs.namespacePrefix, "namespace-prefix", "", "", "Namespace prefix. The scenario resources will be created in the namespaces with the prefix")
	// prepareCmd.PersistentFlags().StringVarP(&generateArgs.namespaceRange, "namespace-range", "", "", "")
	// prepareCmd.PersistentFlags().StringVarP(&generateArgs.namespace, "namespace", "", "", "Namespace name. The scenario resources will be created in the namespace")

	return prepareCmd
}

func NewEventingPrepareListCommand(p *pkg.PerfParams) *cobra.Command {
	var listCmd = &cobra.Command{
		Use:   "list",
		Short: "List prepare scenarios",

		RunE: func(cmd *cobra.Command, args []string) error {
			for name, ctor := range scenario.GetCommandCtors() {
				cmd := ctor(p)

				w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
				fmt.Fprintf(w, "%s\t%s\n", name, cmd.Short)
				w.Flush()
			}
			return nil
		},
	}
	return listCmd
}
