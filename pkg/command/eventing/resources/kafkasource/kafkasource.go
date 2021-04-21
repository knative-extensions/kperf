/*
Copyright 2021 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package kafkasource

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	bindings "knative.dev/eventing-kafka/pkg/apis/bindings/v1beta1"
	sources "knative.dev/eventing-kafka/pkg/apis/sources/v1beta1"
	duckv1 "knative.dev/pkg/apis/duck/v1"

	"knative.dev/kperf/pkg"
)

type Options struct {
	BootstrapServers []string
	Topics           []string
	Sink             duckv1.Destination
}

func Install(p *pkg.PerfParams, name string, namespace string, options Options) error {
	client, err := p.NewKafkaSourceClient()
	if err != nil {
		return err
	}

	source := &sources.KafkaSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-kafka-source",
			Namespace: namespace,
		},
		Spec: sources.KafkaSourceSpec{
			KafkaAuthSpec: bindings.KafkaAuthSpec{
				BootstrapServers: options.BootstrapServers,
			},
			Topics:        options.Topics,
			ConsumerGroup: "test-consumer-group",
			SourceSpec: duckv1.SourceSpec{
				Sink: options.Sink,
			},
		},
	}

	_, err = client.KafkaSources(namespace).Create(context.Background(), source, metav1.CreateOptions{})
	return err
}
