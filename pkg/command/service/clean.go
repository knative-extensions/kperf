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
	"errors"
	"fmt"
	"strconv"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/spf13/cobra"

	"knative.dev/kperf/pkg"
	"knative.dev/kperf/pkg/generator"
)

func NewServiceCleanCommand(p *pkg.PerfParams) *cobra.Command {
	cleanArgs := cleanArgs{}
	ksvcCleanCommand := &cobra.Command{
		Use:   "clean",
		Short: "clean ksvc",
		Long: `clean ksvc workload

For example:
# To clean Knative Service workload
kperf service clean --namespace-prefix testns / --namespace nsname
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var namespaceRangeMap map[string]bool = map[string]bool{}
			if cleanArgs.namespacePrefix != "" {
				r := strings.Split(cleanArgs.namespaceRange, ",")
				if len(r) != 2 {
					return fmt.Errorf("expected range like 1,500, given %s\n", cleanArgs.namespaceRange)
				}
				start, err := strconv.Atoi(r[0])
				if err != nil {
					return err
				}
				end, err := strconv.Atoi(r[1])
				if err != nil {
					return err
				}
				if start >= 0 && end >= 0 && start <= end {
					for i := start; i <= end; i++ {
						namespaceRangeMap[fmt.Sprintf("%s-%d", cleanArgs.namespacePrefix, i)] = true
					}
				} else {
					return fmt.Errorf("failed to parse namespace range %s\n", cleanArgs.namespaceRange)
				}
			}

			ksvcClient, err := p.NewServingClient()
			if err != nil {
				return err
			}
			nsNameList := []string{}
			if cleanArgs.namespace != "" {
				nss, err := p.ClientSet.CoreV1().Namespaces().Get(context.TODO(), cleanArgs.namespace, metav1.GetOptions{})
				if err != nil {
					return err
				}
				nsNameList = append(nsNameList, nss.Name)
			} else if cleanArgs.namespacePrefix != "" {
				nsList, err := p.ClientSet.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
				if err != nil {
					return err
				}
				if len(nsList.Items) == 0 {
					return fmt.Errorf("no namespace found with prefix %s", cleanArgs.namespacePrefix)
				}
				if len(namespaceRangeMap) >= 0 {
					for i := 0; i < len(nsList.Items); i++ {
						if _, exists := namespaceRangeMap[nsList.Items[i].Name]; exists {
							nsNameList = append(nsNameList, nsList.Items[i].Name)
						}
					}
				} else {
					for i := 0; i < len(nsList.Items); i++ {
						if strings.HasPrefix(nsList.Items[i].Name, cleanArgs.namespacePrefix) {
							nsNameList = append(nsNameList, nsList.Items[i].Name)
						}
					}
				}

				if len(nsNameList) == 0 {
					return fmt.Errorf("no namespace found with prefix %s", cleanArgs.namespacePrefix)
				}
			} else {
				return errors.New("both namespace and namespace-prefix are empty")
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
						if strings.HasPrefix(svcList.Items[j].Name, cleanArgs.svcPrefix) {
							matchedNsNameList = append(matchedNsNameList, [2]string{nsNameList[i], svcList.Items[j].Name})
						}
					}
				}
			}
			if len(matchedNsNameList) > 0 {
				generator.NewBatchCleaner(matchedNsNameList, cleanArgs.concurrency, cleanKsvc).Clean()
			} else {
				fmt.Println("No service found for cleaning")
			}
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
