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
	"fmt"

	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"

	"github.com/spf13/cobra"

	"knative.dev/kperf/pkg"
)

func NewEventingCleanCommand(p *pkg.PerfParams) *cobra.Command {
	cleanArgs := cleanArgs{}
	ksvcCleanCommand := &cobra.Command{
		Use:   "clean",
		Short: "Clean eventing workload",
		Long: `Clean eventing workload

For example:
# To clean Knative Eventing workload
kperf eventing clean --namespace-prefix testns / --namespace nsname
`,
		RunE: func(cmd *cobra.Command, args []string) error {

			fmt.Println("Eventng cleaning called")
			return nil
		},
	}

	ksvcCleanCommand.Flags().StringVarP(&cleanArgs.namespacePrefix, "namespace-prefix", "", "", "Namespace prefix. The ksvc in namespaces with the prefix will be cleaned.")
	ksvcCleanCommand.Flags().StringVarP(&cleanArgs.namespaceRange, "namespace-range", "", "", "")
	ksvcCleanCommand.Flags().StringVarP(&cleanArgs.namespace, "namespace", "", "", "Namespace name. The ksvc in the namespace will be cleaned.")
	ksvcCleanCommand.Flags().StringVarP(&cleanArgs.svcPrefix, "svc-prefix", "", "testksvc", "ksvc name prefix. The ksvcs will be svcPrefix1,svcPrefix2,svcPrefix3......")
	ksvcCleanCommand.Flags().IntVarP(&cleanArgs.concurrency, "concurrency", "c", 10, "Number of multiple ksvcs to make at a time")

	return ksvcCleanCommand
}
