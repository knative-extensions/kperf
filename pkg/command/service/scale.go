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
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"

	_ "k8s.io/client-go/plugin/pkg/client/auth"

	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"knative.dev/kperf/pkg"
	"knative.dev/kperf/pkg/command/utils"

	"knative.dev/serving/pkg/apis/serving"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	servingv1client "knative.dev/serving/pkg/client/clientset/versioned/typed/serving/v1"
)

const (
	OutputFilename = "ksvc_scaling_time"
)

type ServicesToScale struct {
	Namespace string
	Service   *servingv1.Service
}

type Response struct {
	Status     string
	StatusCode int
	Header     http.Header
	Body       []byte
}

func NewServiceScaleCommand(p *pkg.PerfParams) *cobra.Command {
	scaleArgs := pkg.ScaleArgs{}
	serviceScaleCommand := &cobra.Command{
		Use:   "scale",
		Short: "Scale and Measure Knative service",
		Long: `Scale Knative service from zero and measure time

For example:
# To measure a Knative Service scaling from zero
kperf service scale --svc-perfix svc --range 1,200 --namespace ns --concurrency 20
`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Flags().NFlag() == 0 {
				return fmt.Errorf("'service scale' requires flag(s)")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return ScaleServicesUpFromZero(p, scaleArgs)
		},
	}

	serviceScaleCommand.Flags().StringVarP(&scaleArgs.SvcRange, "range", "r", "", "Desired service range")
	serviceScaleCommand.Flags().StringVarP(&scaleArgs.Namespace, "namespace", "", "", "Service namespace")
	serviceScaleCommand.Flags().StringVarP(&scaleArgs.SvcPrefix, "svc-prefix", "", "", "Service name prefix")
	serviceScaleCommand.Flags().BoolVarP(&scaleArgs.Verbose, "verbose", "v", false, "Service verbose result")
	serviceScaleCommand.Flags().StringVarP(&scaleArgs.NamespaceRange, "namespace-range", "", "", "Service namespace range")
	serviceScaleCommand.Flags().StringVarP(&scaleArgs.NamespacePrefix, "namespace-prefix", "", "", "Service namespace prefix")
	serviceScaleCommand.Flags().IntVarP(&scaleArgs.Concurrency, "concurrency", "c", 10, "Number of workers to do measurement job")
	serviceScaleCommand.Flags().StringVarP(&scaleArgs.Output, "output", "o", ".", "Measure result location")
	serviceScaleCommand.Flags().BoolVarP(&scaleArgs.ResolvableDomain, "resolvable", "", false, "If Service endpoint resolvable url")
	serviceScaleCommand.Flags().IntVarP(&scaleArgs.MaxRetries, "MaxRetries", "", 10, "Maximum number of trying to poll the service")
	serviceScaleCommand.Flags().DurationVarP(&scaleArgs.RequestInterval, "wait", "", 2*time.Second, "Time to wait before retring to call the Knatice Service")
	serviceScaleCommand.Flags().DurationVarP(&scaleArgs.RequestTimeout, "timeout", "", 2*time.Second, "Duration to wait for Knative Service to be ready")
	return serviceScaleCommand
}

func ScaleServicesUpFromZero(params *pkg.PerfParams, inputs pkg.ScaleArgs) error {
	ctx := context.Background()
	nsNameList, err := GetNamespaces(ctx, params, inputs.Namespace, inputs.NamespaceRange, inputs.NamespacePrefix)
	if err != nil {
		return err
	}

	scaleFromZeroResult, err := scaleAndMeasure(ctx, params, inputs, nsNameList, getServices)
	if err != nil {
		return err
	}

	knativeVersion := GetKnativeVersion(params)
	ingressInfo := GetIngressController(params)
	scaleFromZeroResult.KnativeInfo.ServingVersion = knativeVersion["serving"]
	scaleFromZeroResult.KnativeInfo.EventingVersion = knativeVersion["eventing"]
	scaleFromZeroResult.KnativeInfo.IngressController = ingressInfo["ingressController"]
	scaleFromZeroResult.KnativeInfo.IngressVersion = ingressInfo["version"]

	rows := make([][]string, 0)
	rows = append([][]string{{"svc_name", "svc_namespace", "svc_latency", "deployment_latency"}}, rows...)

	for _, m := range scaleFromZeroResult.Measurment {
		rows = append(rows, []string{m.ServiceName, m.ServiceNamespace, fmt.Sprintf("%f", m.ServiceLatency), fmt.Sprintf("%f", m.DeploymentLatency)})
	}

	current := time.Now()
	outputLocation, err := utils.CheckOutputLocation(inputs.Output)
	if err != nil {
		fmt.Printf("failed to check measure output location: %s\n", err)
	}

	csvPath := filepath.Join(outputLocation, fmt.Sprintf("%s_%s.csv", current.Format(DateFormatString), OutputFilename))
	err = utils.GenerateCSVFile(csvPath, rows)
	if err != nil {
		fmt.Printf("failed to generate CSV file and skip %s\n", err)
	}
	fmt.Printf("Measurement saved in CSV file %s\n", csvPath)

	jsonPath := filepath.Join(outputLocation, fmt.Sprintf("%s_%s.json", current.Format(DateFormatString), OutputFilename))
	jsonData, err := json.Marshal(scaleFromZeroResult)
	if err != nil {
		fmt.Printf("failed to generate json data and skip %s\n", err)
	}
	err = utils.GenerateJSONFile(jsonData, jsonPath)
	if err != nil {
		fmt.Printf("failed to generate json file and skip %s\n", err)
	}
	fmt.Printf("Measurement saved in JSON file %s\n", jsonPath)

	htmlPath := filepath.Join(outputLocation, fmt.Sprintf("%s_%s.html", current.Format(DateFormatString), OutputFilename))
	err = utils.GenerateHTMLFile(csvPath, htmlPath)
	if err != nil {
		fmt.Printf("failed to generate HTML file and skip %s\n", err)
	}
	fmt.Printf("Visualized measurement saved in HTML file %s\n", htmlPath)

	return nil
}

func scaleAndMeasure(ctx context.Context, params *pkg.PerfParams, inputs pkg.ScaleArgs, nsNameList []string, servicesListFunc func(context.Context, servingv1client.ServingV1Interface, []string, string) []ServicesToScale) (pkg.ScaleResult, error) {
	result := pkg.ScaleResult{}
	ksvcClient, err := params.NewServingClient()
	if err != nil {
		return result, err
	}
	objs := servicesListFunc(ctx, ksvcClient, nsNameList, inputs.SvcPrefix)
	count := len(objs)

	var wg sync.WaitGroup
	var m sync.Mutex
	wg.Add(count)
	for i := 0; i < count; i++ {
		go func(ndx int, m *sync.Mutex) {
			defer wg.Done()
			sdur, ddur, err := runScaleFromZero(ctx, params, inputs, objs[ndx].Namespace, objs[ndx].Service)
			if err == nil {
				//measure
				fmt.Printf("result of scale for service %s is %f, %f \n", objs[ndx].Service.Name, sdur.Seconds(), ddur.Seconds())
				m.Lock()
				result.Measurment = append(result.Measurment, pkg.ScaleFromZeroResult{
					ServiceName:       objs[ndx].Service.Name,
					ServiceNamespace:  objs[ndx].Service.Namespace,
					ServiceLatency:    sdur.Seconds(),
					DeploymentLatency: ddur.Seconds(),
				})
				m.Unlock()
			} else {
				fmt.Printf("result of scale is error: %s", err)
			}
		}(i, &m)
	}
	wg.Wait()

	return result, nil
}

func getServices(ctx context.Context, servingClient servingv1client.ServingV1Interface, nsNameList []string, svcPrefix string) []ServicesToScale {
	objs := []ServicesToScale{}
	for _, ns := range nsNameList {
		svcList, err := servingClient.Services(ns).List(ctx, metav1.ListOptions{})
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

func runScaleFromZero(ctx context.Context, params *pkg.PerfParams, inputs pkg.ScaleArgs, namespace string, svc *servingv1.Service) (
	time.Duration, time.Duration, error) {
	selector := labels.SelectorFromSet(labels.Set{
		serving.ServiceLabelKey: svc.Name,
	})

	watcher, err := params.ClientSet.AppsV1().Deployments(namespace).Watch(
		context.Background(), metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		m := fmt.Sprintf("unable to watch the deployment for the service: %v", err)
		log.Println(m)
		return 0, 0, errors.New(m)
	}
	defer watcher.Stop()

	ddch := watcher.ResultChan()
	sdch := make(chan struct{})
	errch := make(chan error)

	endpoint, err := resolveEndpoint(ctx, params, inputs.ResolvableDomain, svc)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get the cluster endpoint: %w", err)
	}

	client := http.Client{}
	req, _ := http.NewRequest("GET", endpoint, nil)
	req.Header.Set("Host", svc.Status.RouteStatusFields.URL.URL().Host)
	go func() {
		_, err = Poll(client, req, inputs.MaxRetries, inputs.RequestInterval, inputs.RequestTimeout, endpoint)
		if err != nil {
			m := fmt.Sprintf("the endpoint for Route %q at %q didn't serve the expected text %v", svc.Name, endpoint, err)
			log.Println(m)
			errch <- errors.New(m)
			return
		}

		sdch <- struct{}{}
	}()

	start := time.Now()
	// Get the duration that takes to change deployment spec.
	var dd time.Duration
	for {
		select {
		case event := <-ddch:
			if event.Type == watch.Modified {
				dm := event.Object.(*v1.Deployment)
				if *dm.Spec.Replicas != 0 && dd == 0 {
					dd = time.Since(start)
				}
			}
		case <-sdch:
			return time.Since(start), dd, nil
		case err := <-errch:
			return 0, 0, err
		}
	}
}

func Poll(httpClient http.Client, request *http.Request, maxRetries int, requestInterval time.Duration, requestTimeout time.Duration, url string) (*Response, error) {
	var resp *Response
	retries := 0
	err := wait.PollImmediate(requestInterval, requestTimeout, func() (bool, error) {
		rawResp, err := httpClient.Do(request)
		if err != nil {
			if retries < maxRetries {
				fmt.Printf("Retrying %s\n", url)
				return false, nil
			}
			fmt.Printf("NOT Retrying %s: %v\n", url, err)
			return true, err
		}

		defer rawResp.Body.Close()
		retries = retries + 1

		body, err := ioutil.ReadAll(rawResp.Body)
		if err != nil {
			return true, err
		}

		resp = &Response{
			Status:     rawResp.Status,
			StatusCode: rawResp.StatusCode,
			Header:     rawResp.Header,
			Body:       body,
		}

		return true, nil
	})

	if err != nil {
		return resp, fmt.Errorf("response did not pass checks %s", err)
	}

	return resp, nil
}

// resolveEndpoint resolves the endpoint address considering whether the domain is resolvable
func resolveEndpoint(ctx context.Context, params *pkg.PerfParams, resolvable bool, svc *servingv1.Service) (string, error) {
	// If the domain is resolvable, it can be used directly
	if resolvable {
		url := svc.Status.RouteStatusFields.URL.URL()
		return url.String(), nil
	}
	// Otherwise, use the actual cluster endpoint
	return getIngressEndpoint(ctx, params)
}

// getIngressEndpoint gets the ingress public IP or hostname.
// address - is the endpoint to which we should actually connect.
// err - an error when address cannot be established.
func getIngressEndpoint(ctx context.Context, params *pkg.PerfParams) (address string, err error) {
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

	endpoint, err := endpointFromService(ingress)
	if err != nil {
		return "", err
	}
	url := url.URL{Scheme: "http", Host: endpoint}
	return url.String(), nil
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
