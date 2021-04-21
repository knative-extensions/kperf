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
package pkg

import (
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	kafkasourcev1beta1 "knative.dev/eventing-kafka/pkg/client/clientset/versioned/typed/sources/v1beta1"
	networkingv1alpha1 "knative.dev/networking/pkg/client/clientset/versioned/typed/networking/v1alpha1"
	autoscalingv1alpha1 "knative.dev/serving/pkg/client/clientset/versioned/typed/autoscaling/v1alpha1"
	servingv1client "knative.dev/serving/pkg/client/clientset/versioned/typed/serving/v1"
)

type PerfParams struct {
	KubeCfgPath          string
	ClientConfig         clientcmd.ClientConfig
	ClientSet            kubernetes.Interface
	NewAutoscalingClient func() (autoscalingv1alpha1.AutoscalingV1alpha1Interface, error)
	NewServingClient     func() (servingv1client.ServingV1Interface, error)
	NewNetworkingClient  func() (networkingv1alpha1.NetworkingV1alpha1Interface, error)
	NewKafkaSourceClient func() (kafkasourcev1beta1.SourcesV1beta1Interface, error)
}

func (params *PerfParams) Initialize() error {
	if params.ClientSet == nil {
		restConfig, err := params.RestConfig()
		if err != nil {
			return err
		}

		params.ClientSet, err = kubernetes.NewForConfig(restConfig)
		if err != nil {
			fmt.Println("failed to create client:", err)
			os.Exit(1)
		}
	}
	if params.NewAutoscalingClient == nil {
		params.NewAutoscalingClient = params.newAutoscalingClient
	}
	if params.NewServingClient == nil {
		params.NewServingClient = params.newServingClient
	}
	if params.NewNetworkingClient == nil {
		params.NewNetworkingClient = params.newNetworkingClient
	}
	if params.NewKafkaSourceClient == nil {
		params.NewKafkaSourceClient = params.newKafkaSourceClient
	}
	return nil
}

func (params *PerfParams) newAutoscalingClient() (autoscalingv1alpha1.AutoscalingV1alpha1Interface, error) {
	restConfig, err := params.RestConfig()
	if err != nil {
		return nil, err
	}
	client, err := autoscalingv1alpha1.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (params *PerfParams) newServingClient() (servingv1client.ServingV1Interface, error) {
	restConfig, err := params.RestConfig()
	if err != nil {
		return nil, err
	}

	client, err := servingv1client.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (params *PerfParams) newNetworkingClient() (networkingv1alpha1.NetworkingV1alpha1Interface, error) {
	restConfig, err := params.RestConfig()
	if err != nil {
		return nil, err
	}

	client, err := networkingv1alpha1.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (params *PerfParams) newKafkaSourceClient() (kafkasourcev1beta1.SourcesV1beta1Interface, error) {
	restConfig, err := params.RestConfig()
	if err != nil {
		return nil, err
	}

	client, err := kafkasourcev1beta1.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}
	return client, nil
}

// RestConfig returns REST config, which can be to use to create specific clientset
func (params *PerfParams) RestConfig() (*rest.Config, error) {
	var err error

	if params.ClientConfig == nil {
		params.ClientConfig, err = params.GetClientConfig()
		if err != nil {
			return nil, err
		}
	}

	config, err := params.ClientConfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	return config, nil
}

// GetClientConfig gets ClientConfig from KubeCfgPath
func (params *PerfParams) GetClientConfig() (clientcmd.ClientConfig, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if len(params.KubeCfgPath) == 0 {
		return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{}), nil
	}

	_, err := os.Stat(params.KubeCfgPath)
	if err == nil {
		loadingRules.ExplicitPath = params.KubeCfgPath
		return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{}), nil
	}

	if !os.IsNotExist(err) {
		return nil, err
	}

	paths := filepath.SplitList(params.KubeCfgPath)
	if len(paths) > 1 {
		return nil, fmt.Errorf("Can not find config file. '%s' looks like a path. "+
			"Please use the env var KUBECONFIG if you want to check for multiple configuration files", params.KubeCfgPath)
	}
	return nil, fmt.Errorf("Config file '%s' can not be found", params.KubeCfgPath)
}
