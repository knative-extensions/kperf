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
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
	knativeapis "knative.dev/pkg/apis"

	networkingv1 "knative.dev/networking/pkg/apis/networking/v1alpha1"
	autoscalingv1 "knative.dev/serving/pkg/apis/autoscaling/v1alpha1"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"

	servingv1client "knative.dev/serving/pkg/client/clientset/versioned/typed/serving/v1"
)

const (
	DefaultNamespace = "default"
	ServiceImage     = "gcr.io/knative-samples/helloworld-go"
)

type ServicesToScale struct {
	Namespace string
	Service   *servingv1.Service
}

type GetServicesFunc func(servingv1client.ServingV1Interface, []string, string) []ServicesToScale

func GetCheckServiceStatusReadyFunc(ksvcClient servingv1client.ServingV1Interface, timeout time.Duration) func(ns, name string) error {
	return func(ns, name string) error {
		start := time.Now()
		for time.Since(start) < timeout {
			svc, _ := ksvcClient.Services(ns).Get(context.TODO(), name, metav1.GetOptions{})
			conditions := svc.Status.Conditions
			for i := 0; i < len(conditions); i++ {
				if conditions[i].Type == knativeapis.ConditionReady && conditions[i].IsTrue() {
					return nil
				}
			}
		}
		fmt.Printf("Error: Knative Service %s in namespace %s is not ready after %s\n", name, ns, timeout)
		return fmt.Errorf("Knative Service %s in namespace %s is not ready after %s ", name, ns, timeout)
	}
}

func GetCreateKsvcFunc(ksvcClient servingv1client.ServingV1Interface, minScale, maxScale int, svcPrefix string, timeout time.Duration) func(ns string, index int) (string, string) {
	return func(ns string, index int) (string, string) {
		service := servingv1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-%d", svcPrefix, index),
				Namespace: ns,
			},
		}

		service.Spec.Template = servingv1.RevisionTemplateSpec{
			Spec: servingv1.RevisionSpec{},
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"autoscaling.knative.dev/minScale": strconv.Itoa(minScale),
					"autoscaling.knative.dev/maxScale": strconv.Itoa(maxScale),
				},
			},
		}
		service.Spec.Template.Spec.Containers = []corev1.Container{
			{
				Image: ServiceImage,
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
}

func GetListServicesFunc() GetServicesFunc {
	return func(servingClient servingv1client.ServingV1Interface, nsNameList []string, svcPrefix string) []ServicesToScale {
		objs := []ServicesToScale{}
		for _, ns := range nsNameList {
			svcList, err := servingClient.Services(ns).List(context.TODO(), metav1.ListOptions{})
			if err == nil {
				for _, s := range svcList.Items {
					svc := s
					if strings.HasPrefix(s.Name, svcPrefix) {
						objs = append(objs, ServicesToScale{Namespace: ns, Service: &svc})
					}
				}
			}
		}
		return objs
	}
}

func GetRevisionReadyCondition() apis.ConditionType {
	return servingv1.RevisionConditionReady
}

func GetServiceConditionConfigurationsReady() apis.ConditionType {
	return servingv1.ServiceConditionConfigurationsReady
}

func GetServiceConditionRoutesReadyType() apis.ConditionType {
	return servingv1.ServiceConditionRoutesReady
}

func GetActivatorEndpointsPopulatedCondition() apis.ConditionType {
	return networkingv1.ActivatorEndpointsPopulated
}
func GetServerlessServiceConditionEndspointsPopulated() apis.ConditionType {
	return networkingv1.ServerlessServiceConditionEndspointsPopulated
}

func GetServerlessServiceConditionReady() apis.ConditionType {
	return networkingv1.ServerlessServiceConditionReady
}

func GetIngressConditionNetworkConfigured() apis.ConditionType {
	return networkingv1.IngressConditionNetworkConfigured
}

func GetIngressConditionLoadBalancerReady() apis.ConditionType {
	return networkingv1.IngressConditionLoadBalancerReady
}

func GetPodAutoscalerConditionActive() apis.ConditionType {
	return autoscalingv1.PodAutoscalerConditionActive
}
