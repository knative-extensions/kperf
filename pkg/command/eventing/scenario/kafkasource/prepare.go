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

package kafkasource

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	duckv1 "knative.dev/pkg/apis/duck/v1"

	"knative.dev/kperf/pkg"
	"knative.dev/kperf/pkg/command/eventing/resources/kafkasource"
	resreceiver "knative.dev/kperf/pkg/command/eventing/resources/receiver"
	"knative.dev/kperf/pkg/command/eventing/util"
)

type KafkaArgs struct {
	BootstrapServers []string
	Topics           []string
	Count            int
}

func NewKafkaSourcePrepareCommand(p *pkg.PerfParams) *cobra.Command {
	kafkaArgs := KafkaArgs{}

	kafkaSourceGenCommand := &cobra.Command{
		Use:   "kafkasource",
		Short: "Prepare the kafkasource scenario",

		RunE: func(cmd *cobra.Command, args []string) error {

			// TODO: support multiple namespaces
			namespace := "kperf-kafkasource"

			_, err := p.ClientSet.CoreV1().Namespaces().Get(context.TODO(), namespace, metav1.GetOptions{})
			if err != nil && apierrors.IsNotFound(err) {
				fmt.Printf("creating namespace %s\n", namespace)
				ns := corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: namespace,
					},
				}
				_, err = p.ClientSet.CoreV1().Namespaces().Create(context.TODO(), &ns, metav1.CreateOptions{})
				if err != nil {
					fmt.Printf("namespace creation failed. (%v)\n", err)
					os.Exit(1)
				}
			} else if err != nil {
				return fmt.Errorf("failed to get namespace: %w", err)
			}

			// Install HTTP receiver
			err = resreceiver.Install(p, "receiver", namespace, resreceiver.Options{})
			if err != nil {
				fmt.Printf("HTTP receiver installation failed. (%v)\n", err)
				os.Exit(1)
			}

			// Install KafkaSources
			for i := 0; i < kafkaArgs.Count; i++ {
				options := kafkasource.Options{
					BootstrapServers: kafkaArgs.BootstrapServers,
					Topics:           kafkaArgs.Topics,
					Sink: duckv1.Destination{
						Ref: &duckv1.KReference{
							APIVersion: "v1",
							Kind:       "Service",
							Name:       "receiver",
							Namespace:  namespace,
						},
					},
				}

				name := util.MakeRandomK8sName("source")
				err = kafkasource.Install(p, name, namespace, options)
				if err != nil {
					fmt.Printf("KafkaSource %s installation failed. (%v)\n", name, err)
					os.Exit(1)
				}
			}

			return nil
		},
	}

	kafkaSourceGenCommand.Flags().StringSliceVarP(&kafkaArgs.BootstrapServers, "brokers", "b", nil, "Bootstrap servers (brokers,...)")
	kafkaSourceGenCommand.Flags().StringSliceVarP(&kafkaArgs.Topics, "topics", "t", nil, "List of topics")
	kafkaSourceGenCommand.Flags().IntVarP(&kafkaArgs.Count, "count", "c", 1, "The number of KafkaSource instances to create per namespace")
	// ksvcGenCommand.Flags().StringVarP(&generateArgs.namespaceRange, "namespace-range", "", "", "")
	// ksvcGenCommand.Flags().StringVarP(&generateArgs.namespace, "namespace", "", "", "Namespace name. The scenario resources will be created in the namespace")

	return kafkaSourceGenCommand
}
