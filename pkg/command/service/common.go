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
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"knative.dev/kperf/pkg"
	"knative.dev/kperf/pkg/command/utils"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	servingv1client "knative.dev/serving/pkg/client/clientset/versioned/typed/serving/v1"
)

type ServicesToScale struct {
	Namespace string
	Service   *servingv1.Service
}

func GetNamespaces(ctx context.Context, params *pkg.PerfParams, namespace, namespaceRange, namespacePrefix string) ([]string, error) {
	nsNameList := []string{}
	var namespaceRangeMap map[string]bool = map[string]bool{}
	if namespacePrefix != "" {
		r := strings.Split(namespaceRange, ",")
		if len(r) != 2 {
			return nsNameList, fmt.Errorf("expected range like 1,500, given %s", namespaceRange)
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
			return nsNameList, fmt.Errorf("failed to parse namespace range %s", namespaceRange)
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

// getServices gets existed services by svc or (svc-prefix, svcRange)
func getServices(ctx context.Context, servingClient servingv1client.ServingV1Interface, nsNameList []string, svcPrefix string, svcRange string, service string) ([]ServicesToScale, error) {
	objs := []ServicesToScale{}
	// generate a svcRange map by svcPrefix and svcRange
	var svcRangeMap map[string]bool = map[string]bool{}
	if svcPrefix != "" && svcRange != "" {
		r := strings.Split(svcRange, ",")
		if len(r) != 2 {
			return objs, fmt.Errorf("expected svc range like 1,500, given %s", svcRange)
		}
		start, err := strconv.Atoi(r[0])
		if err != nil {
			return objs, err
		}
		end, err := strconv.Atoi(r[1])
		if err != nil {
			return objs, err
		}
		if start >= 0 && end >= 0 && start <= end {
			for i := start; i <= end; i++ {
				svcRangeMap[fmt.Sprintf("%s-%d", svcPrefix, i)] = true
			}
		} else {
			return objs, fmt.Errorf("failed to parse svc range %s", svcRange)
		}
	}
	if service != "" { // get existed service by given svc name
		for _, ns := range nsNameList {
			svc, err := servingClient.Services(ns).Get(ctx, service, metav1.GetOptions{})
			if err != nil {
				return objs, err
			}
			if service == svc.Name {
				objs = append(objs, ServicesToScale{Namespace: ns, Service: svc})
			}
		}
		if len(objs) == 0 {
			return objs, fmt.Errorf("svc with name %s not found", service)
		}
	} else if svcPrefix != "" { // get existed services by svcRangeMap and svcPrefix in nsNameList
		for _, ns := range nsNameList {
			svcList, err := servingClient.Services(ns).List(ctx, metav1.ListOptions{})
			if err == nil {
				if len(svcRangeMap) >= 0 { // get existed services in svcRangeMap
					for _, s := range svcList.Items {
						svc := s
						if _, exists := svcRangeMap[svc.Name]; exists {
							objs = append(objs, ServicesToScale{Namespace: ns, Service: &svc})
						}
					}
				} else { // get existed services by svcPrefix if svcRangeMap is empty
					for _, s := range svcList.Items {
						svc := s
						if strings.HasPrefix(s.Name, svcPrefix) {
							objs = append(objs, ServicesToScale{Namespace: ns, Service: &svc})
						}
					}
				}
			}
		}
		if len(objs) == 0 {
			return objs, fmt.Errorf("no ksvc found with prefix %s", svcPrefix)
		}
	} else {
		return objs, errors.New("both svc and svc-prefix are empty")
	}
	return objs, nil
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

// GenerateCSVOutput generates CSV file from the rows data
func GenerateCSVOutput(rows [][]string, outputPathPrefix string) (csvPath string, err error) {
	csvPath = outputPathPrefix + ".csv"
	err = utils.GenerateCSVFile(csvPath, rows)
	if err != nil {
		fmt.Printf("failed to generate CSV file and skip %s\n", err)
		return "", err
	}
	return csvPath, nil
}

// GenerateHTMLOutput generates HTML file from CSV file
func GenerateHTMLOutput(csvPath string, outputPathPrefix string) (htmlPath string, err error) {
	htmlPath = outputPathPrefix + ".html"
	err = utils.GenerateHTMLFile(csvPath, htmlPath)
	if err != nil {
		fmt.Printf("failed to generate HTML file and skip %s\n", err)
		return "", err
	}
	return htmlPath, nil
}

// GenerateJSONOutput generates JSON output from the result
func GenerateJSONOutput(result interface{}, outputPathPrefix string) (jsonPath string, err error) {
	jsonData, err := json.Marshal(result)
	if err != nil {
		fmt.Printf("failed to generate json data and skip %s\n", err)
		return "", err
	}
	jsonPath = outputPathPrefix + ".json"
	err = utils.GenerateJSONFile(jsonData, jsonPath)
	if err != nil {
		fmt.Printf("failed to generate json file and skip %s\n", err)
		return "", err
	}
	return jsonPath, nil
}

// GenerateOutputPathPrefix generates the prefix of output path, which can be combined with a suffix name(.csv) to form a complete path
func GenerateOutputPathPrefix(inputsOutput string, outputFilenameFlag string) (pathPrefix string, err error) {
	current := time.Now()
	outputLocation, err := utils.CheckOutputLocation(inputsOutput)
	if err != nil {
		fmt.Printf("failed to check measure output location: %s\n", err)
		return "", err
	}
	pathPrefix = filepath.Join(outputLocation, fmt.Sprintf("%s_%s", current.Format(DateFormatString), outputFilenameFlag))
	return pathPrefix, nil
}

// GenerateOutput generates outputs according to flags(csvFlag, htmlFlag and josnFlag) from rows and result
func GenerateOutput(inputsOutput string, outputFilenameFlag string, csvFlag bool, htmlFlag bool, jsonFlag bool, rows [][]string, result interface{}) error {
	outputPathPrefix, err := GenerateOutputPathPrefix(inputsOutput, outputFilenameFlag)
	if err != nil {
		return err
	}
	if csvFlag && rows != nil {
		// generate csv file from rows
		csvPath, err := GenerateCSVOutput(rows, outputPathPrefix)
		if err != nil {
			fmt.Printf("failed to save measurement in CSV file: %s\n", err)
			return err
		}
		fmt.Printf("Measurement saved in CSV file %s\n", csvPath)

		// generate html file from csv file
		if htmlFlag && csvPath != "" {
			htmlPath, err := GenerateHTMLOutput(csvPath, outputPathPrefix)
			if err != nil {
				fmt.Printf("failed to save visualized measurement in HTML file: %s\n", err)
				return err
			}
			fmt.Printf("Visualized measurement saved in HTML file %s\n", htmlPath)
		}
	} else if htmlFlag {
		fmt.Printf("HTML output needs CSV output, please reset CSV Flag.\n")
	}
	if jsonFlag {
		// generate json file from result
		jsonPath, err := GenerateJSONOutput(result, outputPathPrefix)
		if err != nil {
			fmt.Printf("failed to save measurement in JSON file: %s\n", err)
			return err
		}
		fmt.Printf("Measurement saved in JSON file %s\n", jsonPath)
	}
	return nil
}

// deleteFile deletes a file from the filepath
func deleteFile(filepath string) error {
	_, err := os.Stat(filepath)
	if err != nil {
		fmt.Printf("stat %s error : %s\n", filepath, err)
		return err
	}
	err = os.Remove(filepath)
	if err != nil {
		fmt.Printf("remove %s error : %s\n", filepath, err)
		return err
	}
	return nil
}

// resolveEndpoint resolves the endpoint address considering whether the domain is resolvable
func resolveEndpoint(ctx context.Context, params *pkg.PerfParams, resolvable bool, https bool, svc *servingv1.Service) (string, error) {
	// If the domain is resolvable, it can be used directly
	if resolvable {
		url := svc.Status.RouteStatusFields.URL.URL()
		return url.String(), nil
	}
	// Otherwise, use the actual cluster endpoint
	return getIngressEndpoint(ctx, params, https)
}

// getIngressEndpoint gets the ingress public IP or hostname.
// address - is the endpoint to which we should actually connect.
// err - an error when address cannot be established.
func getIngressEndpoint(ctx context.Context, params *pkg.PerfParams, https bool) (address string, err error) {
	ingressName := "istio-ingressgateway"
	if gatewayOverride := os.Getenv("GATEWAY_OVERRIDE"); gatewayOverride != "" {
		ingressName = gatewayOverride
	}
	ingressNamespace := "istio-system"
	if gatewayNsOverride := os.Getenv("GATEWAY_NAMESPACE_OVERRIDE"); gatewayNsOverride != "" {
		ingressNamespace = gatewayNsOverride
	}

	ingress, err := params.ClientSet.CoreV1().Services(ingressNamespace).Get(ctx, ingressName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	var endpoint string
	// If the ExternalIP of LoadBalancer is none or pending, get endpoint with node port
	if len(ingress.Status.LoadBalancer.Ingress) == 0 {
		endpoint, err = endpointWithNodePortFromService(ingress, ctx, params, https)
		if err != nil {
			return "", err
		}
	} else {
		endpoint, err = endpointFromService(ingress)
	}

	if err != nil {
		return "", err
	}

	var urlScheme string
	if https {
		urlScheme = "https"
	} else {
		urlScheme = "http"
	}
	endpointURL := url.URL{Scheme: urlScheme, Host: endpoint}
	return endpointURL.String(), nil
}

// endpointFromService extracts the endpoint from the service's ingress.
func endpointFromService(svc *corev1.Service) (string, error) {
	ingresses := svc.Status.LoadBalancer.Ingress
	if len(ingresses) != 1 {
		return "", fmt.Errorf("expected exactly one ingress load balancer, instead had %d: %v", len(ingresses), ingresses)
	}
	itu := ingresses[0]

	switch {
	case itu.IP != "":
		return itu.IP, nil
	case itu.Hostname != "":
		return itu.Hostname, nil
	default:
		return "", fmt.Errorf("expected ingress loadbalancer IP or hostname for %s to be set, instead was empty", svc.Name)
	}
}

// endpointWithNodePortFromService extracts the endpoint consisted host IP and node port from the ingress service
func endpointWithNodePortFromService(svc *corev1.Service, ctx context.Context, params *pkg.PerfParams, https bool) (string, error) {
	ingressPod, err := getIngressPod(ctx, params)
	if err != nil {
		return "", err
	}

	hostIP, err := getHostIPFromPod(ingressPod)
	if err != nil {
		return "", err
	}

	var protcol string
	if https {
		protcol = "https"
	} else {
		protcol = "http2"
	}

	nodePort, err := getNodePortFromService(svc, protcol)
	if err != nil {
		return "", err
	}

	return hostIP + ":" + nodePort, nil
}

// getIngressPod gets one of the ingress Pod.
func getIngressPod(ctx context.Context, params *pkg.PerfParams) (pod *corev1.Pod, err error) {
	ingressNamespace := "istio-system"
	if gatewayNsOverride := os.Getenv("GATEWAY_NAMESPACE_OVERRIDE"); gatewayNsOverride != "" {
		ingressNamespace = gatewayNsOverride
	}

	ingressPodList, err := params.ClientSet.CoreV1().Pods(ingressNamespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	if len(ingressPodList.Items) == 0 {
		return nil, fmt.Errorf("ingress pod list is empty")
	}

	return &ingressPodList.Items[0], nil
}

// getHostIPFromPod gets hostIP of the ingress pod.
func getHostIPFromPod(pod *corev1.Pod) (string, error) {
	hostIP := pod.Status.HostIP
	if hostIP == "" {
		return "", fmt.Errorf("host IP of the ingress pod is empty")
	}
	return hostIP, nil
}

// getNodePort gets node port(http2) from the ingress service.
func getNodePortFromService(svc *corev1.Service, protocol string) (string, error) {
	ingressPorts := svc.Spec.Ports
	if len(ingressPorts) == 0 {
		return "", fmt.Errorf("port list of ingress service is empty")
	}

	for _, port := range ingressPorts {
		if port.Name == protocol {
			return strconv.FormatInt(int64(port.NodePort), 10), nil
		}
	}

	return "", fmt.Errorf("%s port of ingress service not found", protocol)
}
