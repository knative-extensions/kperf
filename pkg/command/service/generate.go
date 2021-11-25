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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/spf13/cobra"

	"knative.dev/kperf/internal"
	"knative.dev/kperf/pkg"
	"knative.dev/kperf/pkg/generator"
)

const (
	DefaultNamespace = "default"
)

func NewServiceGenerateCommand(p *pkg.PerfParams) *cobra.Command {
	generateArgs := pkg.GenerateArgs{}

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
			return GenerateServices(p, generateArgs)
		},
	}
	ksvcGenCommand.Flags().IntVarP(&generateArgs.Number, "number", "n", 0, "Total number of Knative Service to be created")
	ksvcGenCommand.MarkFlagRequired("number")
	ksvcGenCommand.Flags().IntVarP(&generateArgs.Interval, "interval", "i", 0, "Interval for each batch generation")
	ksvcGenCommand.MarkFlagRequired("interval")
	ksvcGenCommand.Flags().IntVarP(&generateArgs.Batch, "batch", "b", 0, "Number of Knative Service each time to be created")
	ksvcGenCommand.MarkFlagRequired("batch")
	ksvcGenCommand.Flags().IntVarP(&generateArgs.Concurrency, "concurrency", "c", 10, "Number of multiple Knative Services to make at a time")
	ksvcGenCommand.Flags().IntVarP(&generateArgs.MinScale, "min-scale", "", 0, "For autoscaling.knative.dev/minScale")
	ksvcGenCommand.Flags().IntVarP(&generateArgs.MaxScale, "max-scale", "", 0, "For autoscaling.knative.dev/minScale")

	ksvcGenCommand.Flags().StringVarP(&generateArgs.NamespacePrefix, "namespace-prefix", "", "", "Namespace prefix. The Knative Services will be created in the namespaces with the prefix")
	ksvcGenCommand.Flags().StringVarP(&generateArgs.NamespaceRange, "namespace-range", "", "", "")
	ksvcGenCommand.Flags().StringVarP(&generateArgs.Namespace, "namespace", "", "", "Namespace name. The Knative Services will be created in the namespace")

	ksvcGenCommand.Flags().StringVarP(&generateArgs.SvcPrefix, "svc-prefix", "", "ksvc", "Knative Service name prefix. The Knative Services will be ksvc-1,ksvc-2,ksvc-3 and etc.")
	ksvcGenCommand.Flags().BoolVarP(&generateArgs.CheckReady, "wait", "", false, "Whether to wait the previous Knative Service to be ready")
	ksvcGenCommand.Flags().DurationVarP(&generateArgs.Timeout, "timeout", "", 10*time.Minute, "Duration to wait for previous Knative Service to be ready")

	return ksvcGenCommand
}

// GenerateServices used to generate Knative Service workload
func GenerateServices(params *pkg.PerfParams, inputs pkg.GenerateArgs) error {
	nsNameList := []string{}
	if inputs.NamespacePrefix == "" && inputs.Namespace == "" {
		nsNameList = []string{DefaultNamespace}
	} else if inputs.NamespacePrefix != "" {
		r := strings.Split(inputs.NamespaceRange, ",")
		if len(r) != 2 {
			return fmt.Errorf("expected range like 1,500, given %s\n", inputs.NamespaceRange)
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
				nsNameList = append(nsNameList, fmt.Sprintf("%s-%d", inputs.NamespacePrefix, i))
			}
		} else {
			return errors.New("failed to parse namespace range")
		}
	} else if inputs.Namespace != "" {
		nsNameList = append(nsNameList, inputs.Namespace)
	}

	// Check if namespace exists, in NOT, return error
	for _, ns := range nsNameList {
		_, err := params.ClientSet.CoreV1().Namespaces().Get(context.TODO(), ns, metav1.GetOptions{})
		if err != nil && apierrors.IsNotFound(err) {
			return fmt.Errorf("namespace %s not found, please create one", ns)
		} else if err != nil {
			return fmt.Errorf("failed to get namespace: %w", err)
		}
	}

	ksvcClient, err := params.KnClients.ServingClient()
	if err != nil {
		return err
	}
	createKSVCFunc := internal.GetCreateKsvcFunc(ksvcClient, inputs.MinScale, inputs.MaxScale, inputs.SvcPrefix, inputs.Timeout)
	checkServiceStatusReadyFunc := internal.GetCheckServiceStatusReadyFunc(ksvcClient, inputs.Timeout)

	if inputs.CheckReady {
		generator.NewBatchGenerator(time.Duration(inputs.Interval)*time.Second, inputs.Number, inputs.Batch, inputs.Concurrency, nsNameList, createKSVCFunc, checkServiceStatusReadyFunc).Generate()
	} else {
		generator.NewBatchGenerator(time.Duration(inputs.Interval)*time.Second, inputs.Number, inputs.Batch, inputs.Concurrency, nsNameList, createKSVCFunc, func(ns, name string) error { return nil }).Generate()
	}

	return nil
}
