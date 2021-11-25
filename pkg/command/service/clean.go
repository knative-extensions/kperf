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

package service

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/spf13/cobra"

	"knative.dev/kperf/pkg"
	"knative.dev/kperf/pkg/generator"
)

func NewServiceCleanCommand(p *pkg.PerfParams) *cobra.Command {
	cleanArgs := pkg.CleanArgs{}
	ksvcCleanCommand := &cobra.Command{
		Use:   "clean",
		Short: "clean ksvc",
		Long: `clean ksvc workload

For example:
# To clean Knative Service workload
kperf service clean --namespace-prefix testns / --namespace nsname
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return CleanServices(p, cleanArgs)
		},
	}

	ksvcCleanCommand.Flags().StringVarP(&cleanArgs.NamespacePrefix, "namespace-prefix", "", "", "Namespace prefix. The ksvc in namespaces with the prefix will be cleaned.")
	ksvcCleanCommand.Flags().StringVarP(&cleanArgs.NamespaceRange, "namespace-range", "", "", "")
	ksvcCleanCommand.Flags().StringVarP(&cleanArgs.Namespace, "namespace", "", "", "Namespace name. The ksvc in the namespace will be cleaned.")
	ksvcCleanCommand.Flags().StringVarP(&cleanArgs.SvcPrefix, "svc-prefix", "", "testksvc", "ksvc name prefix. The ksvcs will be svcPrefix1,svcPrefix2,svcPrefix3......")
	ksvcCleanCommand.Flags().IntVarP(&cleanArgs.Concurrency, "concurrency", "c", 10, "Number of multiple ksvcs to make at a time")

	return ksvcCleanCommand
}

// CleanServices used to clean Knative Service workload
func CleanServices(params *pkg.PerfParams, inputs pkg.CleanArgs) error {
	nsNameList, err := GetNamespaces(context.Background(), params, inputs.Namespace, inputs.NamespaceRange, inputs.NamespacePrefix)
	if err != nil {
		return err
	}

	ksvcClient, err := params.KnClients.ServingClient()
	if err != nil {
		return err
	}

	matchedNsNameList := [][2]string{}
	cleanKsvc := func(namespace, name string) {
		fmt.Printf("Delete ksvc %s in namespace %s\n", name, namespace)
		err := ksvcClient.Services(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
		if err != nil {
			fmt.Printf("Failed to delete ksvc %s in namespace %s\n", name, namespace)
		}
	}
	for i := 0; i < len(nsNameList); i++ {
		svcList, err := ksvcClient.Services(nsNameList[i]).List(context.TODO(), metav1.ListOptions{})
		if err == nil {
			for j := 0; j < len(svcList.Items); j++ {
				if strings.HasPrefix(svcList.Items[j].Name, inputs.SvcPrefix) {
					matchedNsNameList = append(matchedNsNameList, [2]string{nsNameList[i], svcList.Items[j].Name})
				}
			}
		}
	}
	if len(matchedNsNameList) > 0 {
		generator.NewBatchCleaner(matchedNsNameList, inputs.Concurrency, cleanKsvc).Clean()
	} else {
		fmt.Println("No service found for cleaning")
	}
	return nil
}
