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
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/spf13/cobra"

	_ "k8s.io/client-go/plugin/pkg/client/auth"

	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"

	"knative.dev/kperf/pkg"

	"knative.dev/serving/pkg/apis/serving"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	servingv1client "knative.dev/serving/pkg/client/clientset/versioned/typed/serving/v1"
)

const (
	OutputFilename = "ksvc_scaling_time"
	InitialScale   = "0"
)

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

	serviceScaleCommand.Flags().StringVarP(&scaleArgs.Svc, "svc", "", "", "Service name")
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
	serviceScaleCommand.Flags().BoolVarP(&scaleArgs.Https, "https", "", false, "Use https with TLS")
	serviceScaleCommand.Flags().IntVarP(&scaleArgs.Iterations, "iterations", "i", 1, "Number of iterations to invoke the service")
	serviceScaleCommand.Flags().DurationVarP(&scaleArgs.TimeInterval, "time-interval", "T", 10*time.Second, "The time interval of each scale up, recommend to set it no less than the sum of the stable window and cold startup time")
	serviceScaleCommand.Flags().StringVarP(&scaleArgs.StableWindow, "stable-window", "s", "6s", "stable window per revision")
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
	rows = append([][]string{{
		"svc_name", "svc_namespace",
		"svc_latency_avg", "svc_latency_min", "svc_latency_max",
		"svc_latency_p50", "svc_latency_p90", "svc_latency_p95", "svc_latency_p99",
		"deployment_latency_avg", "deployment_latency_min", "deployment_latency_max",
		"deployment_latency_p50", "deployment_latency_p90", "deployment_latency_p95", "deployment_latency_p99"}},
		rows...)

	for _, m := range scaleFromZeroResult.Measurment {
		rows = append(rows, []string{
			m.ServiceName, m.ServiceNamespace,
			fmt.Sprintf("%f", m.ServiceLatency.Average), fmt.Sprintf("%f", m.ServiceLatency.Min), fmt.Sprintf("%f", m.ServiceLatency.Max),
			fmt.Sprintf("%f", m.ServiceLatency.P50), fmt.Sprintf("%f", m.ServiceLatency.P90), fmt.Sprintf("%f", m.ServiceLatency.P95), fmt.Sprintf("%f", m.ServiceLatency.P99),
			fmt.Sprintf("%f", m.DeploymentLatency.Average), fmt.Sprintf("%f", m.DeploymentLatency.Min), fmt.Sprintf("%f", m.DeploymentLatency.Max),
			fmt.Sprintf("%f", m.DeploymentLatency.P50), fmt.Sprintf("%f", m.DeploymentLatency.P90), fmt.Sprintf("%f", m.DeploymentLatency.P95), fmt.Sprintf("%f", m.DeploymentLatency.P99)})
	}

	// generate CSV, HTML and JSON outputs from rows and scaleFromZeroResult
	err = GenerateOutput(inputs.Output, OutputFilename, true, true, true, rows, scaleFromZeroResult)
	if err != nil {
		fmt.Printf("failed to generate output: %s\n", err)
		return err
	}

	return nil
}

func scaleAndMeasure(ctx context.Context, params *pkg.PerfParams, inputs pkg.ScaleArgs, nsNameList []string, servicesListFunc func(context.Context, servingv1client.ServingV1Interface, []string, string, string, string) ([]ServicesToScale, error)) (pkg.ScaleResult, error) {
	result := pkg.ScaleResult{}
	ksvcClient, err := params.NewServingClient()
	if err != nil {
		return result, err
	}
	objs, err := servicesListFunc(ctx, ksvcClient, nsNameList, inputs.SvcPrefix, inputs.SvcRange, inputs.Svc)
	if err != nil {
		return result, err
	}
	count := len(objs)
	// update configmap-autosacle, set allow-zero-initial-scale to true
	origin, err := updateAllowZeroInitialScale(ctx, params, "knative-serving", "true")
	if err != nil {
		fmt.Printf("failed to set allow-zero-initial-scale: %s", err)
		return result, err
	}
	// restore configmap
	defer updateAllowZeroInitialScale(ctx, params, "knative-serving", origin)

	var wg sync.WaitGroup
	var m sync.Mutex
	wg.Add(count)
	for i := 0; i < count; i++ {
		go func(ndx int, m *sync.Mutex) {
			defer wg.Done()
			// set stable window and initial scale to speed up scaling to zero
			err := updateKsvc(ctx, ksvcClient, objs[ndx].Namespace, objs[ndx].Service.Name, inputs.StableWindow, InitialScale)
			if err != nil {
				fmt.Printf("failed to set stable window: %s", err)
				return
			}

			fmt.Printf("scale up service %s/%s in %d iterations:\n", objs[ndx].Namespace, objs[ndx].Service.Name, inputs.Iterations)
			var svcLatencyList, dpLatencyList []float64

			// Iterate inputs.Iterations times to get latency(average, max, min, p50...) of scaling service up from zero
			for j := 0; j < inputs.Iterations; j++ {
				time.Sleep(inputs.TimeInterval)
				sdur, ddur, err := runScaleFromZero(ctx, params, inputs, objs[ndx].Namespace, objs[ndx].Service)
				if err == nil {
					svcLatencyList = append(svcLatencyList, sdur.Seconds())
					dpLatencyList = append(dpLatencyList, ddur.Seconds())
				} else {
					fmt.Printf("result of scale is error: %s", err)
					return
				}
			}
			fmt.Printf("====================== service %s/%s result =====================\n", objs[ndx].Namespace, objs[ndx].Service.Name)
			if inputs.Verbose {
				for k := 0; k < len(svcLatencyList); k++ {
					fmt.Printf("iteration %4d, service latency: %f s, deployment latency: %f s\n", k, svcLatencyList[k], dpLatencyList[k])
				}
			}
			fmt.Printf("service latency result:\n")
			svcLatencyResult := latencyResultHandler(svcLatencyList)
			fmt.Printf("deployment latency result:\n")
			dpLatencyResult := latencyResultHandler(dpLatencyList)

			m.Lock()
			result.Measurment = append(result.Measurment, pkg.ScaleFromZeroResult{
				ServiceName:       objs[ndx].Service.Name,
				ServiceNamespace:  objs[ndx].Service.Namespace,
				ServiceLatency:    svcLatencyResult,
				DeploymentLatency: dpLatencyResult,
			})
			m.Unlock()
		}(i, &m)
	}
	wg.Wait()

	return result, nil
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

	endpoint, err := resolveEndpoint(ctx, params, inputs.ResolvableDomain, inputs.Https, svc)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get the cluster endpoint: %w", err)
	}

	client := http.Client{}
	req, _ := http.NewRequest("GET", endpoint, nil)
	req.Host = svc.Status.RouteStatusFields.URL.URL().Host

	start := time.Now()
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

		body, err := io.ReadAll(rawResp.Body)
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
