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
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	knativeapis "knative.dev/pkg/apis"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"

	"github.com/spf13/cobra"

	"knative.dev/kperf/pkg"
	"knative.dev/kperf/pkg/generator"
)

func NewServiceGenerateCommand(p *pkg.PerfParams) *cobra.Command {
	generateArgs := generateArgs{}

	ksvcGenCommand := &cobra.Command{
		Use:   "generate",
		Short: "generate Knative Service",
		Long: `generate Knative Service workload
For example:
# To generate Knative Service workload
kperf service generate -n 500 --interval 20 --batch 20 --min-scale 0 --max-scale 5 (--namespace-prefix testns/ --namespace nsname)
`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			flags := cmd.Flags()
			if flags.Changed("namespace-prefix") && flags.Changed("namespace") {
				return errors.New("expected either namespace with prefix & range or only namespace name")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			nsNameList := []string{}
			if generateArgs.namespacePrefix == "" && generateArgs.namespace == "" {
				nsNameList = []string{"default"}
			} else if generateArgs.namespacePrefix != "" {
				r := strings.Split(generateArgs.namespaceRange, ",")
				if len(r) != 2 {
					return fmt.Errorf("expected range like 1,500, given %s\n", generateArgs.namespaceRange)
				}
				start, err := strconv.Atoi(r[0])
				if err != nil {
					return err
				}
				end, err := strconv.Atoi(r[1])
				if err != nil {
					return err
				}
				if start > 0 && end > 0 && start <= end {
					for i := start; i <= end; i++ {
						nsNameList = append(nsNameList, fmt.Sprintf("%s-%d", generateArgs.namespacePrefix, i))
					}
				} else {
					return errors.New("failed to parse namespace range")
				}
			} else if generateArgs.namespace != "" {
				nsNameList = append(nsNameList, generateArgs.namespace)
			}

			// Check if namespace exists, in NOT, return error
			for _, ns := range nsNameList {
				_, err := p.ClientSet.CoreV1().Namespaces().Get(context.TODO(), ns, metav1.GetOptions{})
				if err != nil && apierrors.IsNotFound(err) {
					return fmt.Errorf("namespace %s not found, please create one", ns)
				} else if err != nil {
					return fmt.Errorf("failed to get namespace: %w", err)
				}
			}

			ksvcClient, err := p.NewServingClient()
			if err != nil {
				return err
			}
			createKSVCFunc := func(ns string, index int) (string, string) {
				service := servingv1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("%s-%d", generateArgs.svcPrefix, index),
						Namespace: ns,
					},
				}

				service.Spec.Template = servingv1.RevisionTemplateSpec{
					Spec: servingv1.RevisionSpec{},
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							"autoscaling.knative.dev/minScale": strconv.Itoa(generateArgs.minScale),
							"autoscaling.knative.dev/maxScale": strconv.Itoa(generateArgs.maxScale),
						},
					},
				}
				service.Spec.Template.Spec.Containers = []corev1.Container{
					{
						Image: "gcr.io/knative-samples/helloworld-go",
						Ports: []corev1.ContainerPort{
							{
								ContainerPort: 8080,
							},
						},
					},
				}
				fmt.Printf("Creating Knative Service %s in namespace %s\n", service.GetName(), service.GetNamespace())
				_, err := ksvcClient.Services(ns).Create(context.TODO(), &service, metav1.CreateOptions{})
				if err != nil {
					fmt.Printf("failed to create Knative Service %s in namespace %s : %s\n", service.GetName(), service.GetNamespace(), err)
				}
				return service.GetNamespace(), service.GetName()
			}
			checkServiceStatusReadyFunc := func(ns, name string) error {
				start := time.Now()
				for time.Since(start) < generateArgs.timeout {
					svc, _ := ksvcClient.Services(ns).Get(context.TODO(), name, metav1.GetOptions{})
					conditions := svc.Status.Conditions
					for i := 0; i < len(conditions); i++ {
						if conditions[i].Type == knativeapis.ConditionReady && conditions[i].IsTrue() {
							return nil
						}
					}
				}
				fmt.Printf("Error: Knative Service %s in namespace %s is not ready after %s\n", name, ns, generateArgs.timeout)
				return fmt.Errorf("Knative Service %s in namespace %s is not ready after %s ", name, ns, generateArgs.timeout)

			}
			if generateArgs.checkReady {
				generator.NewBatchGenerator(time.Duration(generateArgs.interval)*time.Second, generateArgs.number, generateArgs.batch, generateArgs.concurrency, nsNameList, createKSVCFunc, checkServiceStatusReadyFunc).Generate()
			} else {
				generator.NewBatchGenerator(time.Duration(generateArgs.interval)*time.Second, generateArgs.number, generateArgs.batch, generateArgs.concurrency, nsNameList, createKSVCFunc, func(ns, name string) error { return nil }).Generate()
			}

			return nil
		},
	}
	ksvcGenCommand.Flags().IntVarP(&generateArgs.number, "number", "n", 0, "Total number of Knative Service to be created")
	ksvcGenCommand.MarkFlagRequired("number")
	ksvcGenCommand.Flags().IntVarP(&generateArgs.interval, "interval", "i", 0, "Interval for each batch generation")
	ksvcGenCommand.MarkFlagRequired("interval")
	ksvcGenCommand.Flags().IntVarP(&generateArgs.batch, "batch", "b", 0, "Number of Knative Service each time to be created")
	ksvcGenCommand.MarkFlagRequired("batch")
	ksvcGenCommand.Flags().IntVarP(&generateArgs.concurrency, "concurrency", "c", 10, "Number of multiple Knative Services to make at a time")
	ksvcGenCommand.Flags().IntVarP(&generateArgs.minScale, "min-scale", "", 0, "For autoscaling.knative.dev/minScale")
	ksvcGenCommand.Flags().IntVarP(&generateArgs.maxScale, "max-scale", "", 0, "For autoscaling.knative.dev/minScale")

	ksvcGenCommand.Flags().StringVarP(&generateArgs.namespacePrefix, "namespace-prefix", "", "", "Namespace prefix. The Knative Services will be created in the namespaces with the prefix")
	ksvcGenCommand.Flags().StringVarP(&generateArgs.namespaceRange, "namespace-range", "", "", "")
	ksvcGenCommand.Flags().StringVarP(&generateArgs.namespace, "namespace", "", "", "Namespace name. The Knative Services will be created in the namespace")

	ksvcGenCommand.Flags().StringVarP(&generateArgs.svcPrefix, "svc-prefix", "", "ksvc", "Knative Service name prefix. The Knative Services will be ksvc-1,ksvc-2,ksvc-3 and etc.")
	ksvcGenCommand.Flags().BoolVarP(&generateArgs.checkReady, "wait", "", false, "Whether to wait the previous Knative Service to be ready")
	ksvcGenCommand.Flags().DurationVarP(&generateArgs.timeout, "timeout", "", 10*time.Minute, "Duration to wait for previous Knative Service to be ready")

	return ksvcGenCommand
}
