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

package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"knative.dev/kperf/pkg"
)

func GetNamespaces(ctx context.Context, params *pkg.PerfParams, namespace, namespaceRange, namespacePrefix string) ([]string, error) {
	nsNameList := []string{}
	var namespaceRangeMap map[string]bool = map[string]bool{}
	if namespacePrefix != "" {
		r := strings.Split(namespaceRange, ",")
		if len(r) != 2 {
			return nsNameList, fmt.Errorf("expected range like 1,500, given %s\n", namespaceRange)
		}
		start, err := strconv.Atoi(r[0])
		if err != nil {
			return nsNameList, err
		}
		end, err := strconv.Atoi(r[1])
		if err != nil {
			return nsNameList, err
		}
		if start >= 0 && end >= 0 && start <= end {
			for i := start; i <= end; i++ {
				namespaceRangeMap[fmt.Sprintf("%s-%d", namespacePrefix, i)] = true
			}
		} else {
			return nsNameList, fmt.Errorf("failed to parse namespace range %s\n", namespaceRange)
		}
	}

	if namespace != "" {
		nss, err := params.ClientSet.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
		if err != nil {
			return nsNameList, err
		}
		nsNameList = append(nsNameList, nss.Name)
	} else if namespacePrefix != "" {
		nsList, err := params.ClientSet.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
		if err != nil {
			return nsNameList, err
		}
		if len(nsList.Items) == 0 {
			return nsNameList, fmt.Errorf("no namespace found with prefix %s", namespacePrefix)
		}
		if len(namespaceRangeMap) >= 0 {
			for i := 0; i < len(nsList.Items); i++ {
				if _, exists := namespaceRangeMap[nsList.Items[i].Name]; exists {
					nsNameList = append(nsNameList, nsList.Items[i].Name)
				}
			}
		} else {
			for i := 0; i < len(nsList.Items); i++ {
				if strings.HasPrefix(nsList.Items[i].Name, namespacePrefix) {
					nsNameList = append(nsNameList, nsList.Items[i].Name)
				}
			}
		}

		if len(nsNameList) == 0 {
			return nsNameList, fmt.Errorf("no namespace found with prefix %s", namespacePrefix)
		}
	} else {
		return nsNameList, errors.New("both namespace and namespace-prefix are empty")
	}
	return nsNameList, nil
}

// Get Knative Serving and Eventing version
// Returns a map like {"eventing":"0.20.0", "serving":"0.20.0"}
func GetKnativeVersion(p *pkg.PerfParams) map[string]string {
	knativeVersion := make(map[string]string)
	knativeServingNs, err := p.ClientSet.CoreV1().Namespaces().Get(context.TODO(), "knative-serving", metav1.GetOptions{})
	if err != nil {
		fmt.Printf("failed to get Knative Serving version: %s\n", err)
		knativeVersion["serving"] = "Unknown"
	} else {
		servingVersion := knativeServingNs.Labels["serving.knative.dev/release"]
		servingVersion = strings.Trim(servingVersion, "v")
		knativeVersion["serving"] = servingVersion
	}

	knativeEventingNs, err := p.ClientSet.CoreV1().Namespaces().Get(context.TODO(), "knative-eventing", metav1.GetOptions{})
	if err != nil {
		fmt.Printf("failed to get Knative Eventing version: %s\n", err)
		knativeVersion["eventing"] = "Unknown"
	} else {
		eventingVersion := knativeEventingNs.Labels["eventing.knative.dev/release"]
		eventingVersion = strings.Trim(eventingVersion, "v")
		knativeVersion["eventing"] = eventingVersion
	}
	return knativeVersion
}

// Get Knative ingress controller solution and version
// Returns a map like {"ingressController":"Istio", "version":"1.7.3"}
// For now, kperf only support Istio.
// 1) If it is using Istio, get version from istio deployment labels in istio-system.
// 2) If it is using other options, put version as "Unknown".
func GetIngressController(p *pkg.PerfParams) map[string]string {
	ingressController := make(map[string]string)
	knativeServingConfig, err := p.ClientSet.CoreV1().ConfigMaps("knative-serving").Get(context.TODO(), "config-network", metav1.GetOptions{})
	if err != nil {
		fmt.Printf("failed to get Knative ingress controller info: %s\n", err)
		ingressController["ingressController"] = "Unknown"
		ingressController["version"] = "Unknown"
		return ingressController
	}
	ingressClass := knativeServingConfig.Data["ingress.class"]
	if strings.Contains(ingressClass, "istio") {
		ingressController["ingressController"] = "Istio"
		istioVersion, err := p.ClientSet.CoreV1().ConfigMaps("istio-system").Get(context.TODO(), "istio", metav1.GetOptions{})
		if err != nil {
			fmt.Printf("failed to get Istio version: %s\n", err)
			ingressController["version"] = "Unknown"
			return ingressController
		}
		ingressController["version"] = istioVersion.Labels["operator.istio.io/version"]
		return ingressController
	}
	ingressController["ingressController"] = "Unknown"
	ingressController["version"] = "Unknown"
	return ingressController
}
