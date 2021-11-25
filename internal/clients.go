// Copyright 2021 The Knative Authors
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
package internal

import (
	"errors"

	"k8s.io/client-go/rest"
	networkingv1alpha1 "knative.dev/networking/pkg/client/clientset/versioned/typed/networking/v1alpha1"
	autoscalingv1alpha1 "knative.dev/serving/pkg/client/clientset/versioned/typed/autoscaling/v1alpha1"
	servingv1client "knative.dev/serving/pkg/client/clientset/versioned/typed/serving/v1"
)

type PerfParamsClients struct {
	Config            *rest.Config
	AutoscalingClient func() (autoscalingv1alpha1.AutoscalingV1alpha1Interface, error)
	ServingClient     func() (servingv1client.ServingV1Interface, error)
	NetworkingClient  func() (networkingv1alpha1.NetworkingV1alpha1Interface, error)
}

func (pc *PerfParamsClients) NewAutoscalingClient() (autoscalingv1alpha1.AutoscalingV1alpha1Interface, error) {
	if pc.Config == nil {
		return nil, errors.New("RestConfig required for PerfParams cliengts")
	}
	client, err := autoscalingv1alpha1.NewForConfig(pc.Config)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (pc *PerfParamsClients) NewServingClient() (servingv1client.ServingV1Interface, error) {
	if pc.Config == nil {
		return nil, errors.New("RestConfig required for PerfParams cliengts")
	}

	client, err := servingv1client.NewForConfig(pc.Config)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (pc *PerfParamsClients) NewNetworkingClient() (networkingv1alpha1.NetworkingV1alpha1Interface, error) {
	if pc.Config == nil {
		return nil, errors.New("RestConfig required for PerfParams cliengts")
	}

	client, err := networkingv1alpha1.NewForConfig(pc.Config)
	if err != nil {
		return nil, err
	}
	return client, nil
}
