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

package receiver

import (
	corev1 "k8s.io/api/core/v1"
	"knative.dev/kperf/pkg"
	"knative.dev/kperf/pkg/command/eventing/receiver"
)

var (
	labels = map[string]string{
		"app": "kperf-eventing-receiver",
	}
)

type Options struct {
	receiver.KafkaEnvConfig
}

func Install(p *pkg.PerfParams, name string, namespace string, options Options) error {
	deploymentOptions := deploymentOptions{
		Env: []corev1.EnvVar{
			{
				Name:  "KAFKA_BOOTSTRAP_SERVERS",
				Value: options.KafkaServer,
			},
			{
				Name:  "KAFKA_TOPIC",
				Value: options.Topic,
			},
			{
				Name:  "KAFKA_GROUP",
				Value: options.Group,
			},
		},
	}

	err := installDeployment(p, name, namespace, deploymentOptions)
	if err != nil {
		return err
	}

	err = installService(p, name, namespace)
	return err
}
