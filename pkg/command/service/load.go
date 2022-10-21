// Copyright 2022 The Knative Authors
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
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	"log"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
	vegeta "github.com/tsenart/vegeta/v12/lib"

	_ "k8s.io/client-go/plugin/pkg/client/auth"

	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"k8s.io/apimachinery/pkg/watch"

	"knative.dev/kperf/pkg"
	"knative.dev/kperf/pkg/config"
	"knative.dev/serving/pkg/apis/serving"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	servingv1client "knative.dev/serving/pkg/client/clientset/versioned/typed/serving/v1"
)

const (
	LoadOutputFilename = "ksvc_loading_time"
)

func NewServiceLoadCommand(p *pkg.PerfParams) *cobra.Command {
	loadArgs := pkg.LoadArgs{}
	serviceLoadCommand := &cobra.Command{
		Use:   "load",
		Short: "Load test and Measure Knative service",
		Long: `Scale Knative service from zero using load test tool and measure latency for service to scale form 0 to N

For example:
# To measure a Knative Service scaling from zero to N
kperf service load --namespace ktest --svc-prefix ktest --range 0,3 --load-tool wrk --load-duration 60s --load-concurrency 40 --verbose --output /tmp`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			err := config.BindFlags(cmd, "service.load.", nil)
			if err != nil {
				return err
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return LoadServicesUpFromZero(p, loadArgs)
		},
	}

	serviceLoadCommand.Flags().StringVarP(&loadArgs.Namespace, "namespace", "", "", "Service namespace")
	serviceLoadCommand.Flags().StringVarP(&loadArgs.NamespaceRange, "namespace-range", "", "", "Service namespace range")
	serviceLoadCommand.Flags().StringVarP(&loadArgs.NamespacePrefix, "namespace-prefix", "", "", "Service namespace prefix")
	serviceLoadCommand.Flags().StringVarP(&loadArgs.Svc, "svc", "", "", "Service name")
	serviceLoadCommand.Flags().StringVarP(&loadArgs.SvcPrefix, "svc-prefix", "", "", "Service name prefix")
	serviceLoadCommand.Flags().StringVarP(&loadArgs.SvcRange, "range", "r", "", "Desired service range")
	serviceLoadCommand.Flags().BoolVarP(&loadArgs.Verbose, "verbose", "v", false, "Service verbose result")
	serviceLoadCommand.Flags().BoolVarP(&loadArgs.ResolvableDomain, "resolvable", "", false, "If Service endpoint resolvable url")
	serviceLoadCommand.Flags().DurationVarP(&loadArgs.WaitPodsReadyDuration, "wait-time", "w", 10*time.Second, "Time to wait for all pods to be ready")
	serviceLoadCommand.Flags().StringVarP(&loadArgs.LoadTool, "load-tool", "t", "default", "Select the load test tool, use internal load test tool(vegeta) by default, also support external load tool(wrk and hey, require preinstallation)")
	serviceLoadCommand.Flags().StringVarP(&loadArgs.LoadConcurrency, "load-concurrency", "c", "30", "total number of workers to run concurrently for the load test tool")
	serviceLoadCommand.Flags().StringVarP(&loadArgs.LoadDuration, "load-duration", "d", "60s", "Duration of the test for the load test tool")
	serviceLoadCommand.Flags().StringVarP(&loadArgs.Output, "output", "o", ".", "Measure result location")
	serviceLoadCommand.Flags().BoolVarP(&loadArgs.Https, "https", "", false, "Use https with TLS")

	return serviceLoadCommand
}

func LoadServicesUpFromZero(params *pkg.PerfParams, inputs pkg.LoadArgs) error {
	ctx := context.Background()
	nsNameList, err := GetNamespaces(ctx, params, inputs.Namespace, inputs.NamespaceRange, inputs.NamespacePrefix)
	if err != nil {
		return err
	}
	loadFromZeroResult, err := loadAndMeasure(ctx, params, inputs, nsNameList, getServices)
	if err != nil {
		return err
	}

	knativeVersion := GetKnativeVersion(params)
	ingressInfo := GetIngressController(params)
	loadFromZeroResult.KnativeInfo.ServingVersion = knativeVersion["serving"]
	loadFromZeroResult.KnativeInfo.EventingVersion = knativeVersion["eventing"]
	loadFromZeroResult.KnativeInfo.IngressController = ingressInfo["ingressController"]
	loadFromZeroResult.KnativeInfo.IngressVersion = ingressInfo["version"]

	rows := make([][]string, 0) // replicas ready duration of all services

	maxReplicasCount, replicasCountList := getReplicasCount(loadFromZeroResult)

	//Add replicas ready duration results of each service to rows, only select results before the replicas reaches the maximum count
	for i := 0; i < len(loadFromZeroResult.Measurment); i++ {
		var row []string
		row = append(row, loadFromZeroResult.Measurment[i].ServiceName, loadFromZeroResult.Measurment[i].ServiceNamespace)
		for j := 0; j < len(loadFromZeroResult.Measurment[i].ReplicaResults); j++ {
			if loadFromZeroResult.Measurment[i].ReplicaResults[j].ReadyReplicasCount <= replicasCountList[i] {
				row = append(row, strconv.FormatFloat(loadFromZeroResult.Measurment[i].ReplicaResults[j].ReplicaReadyDuration, 'f', 3, 32))
			}
		}
		rows = append(rows, row)
	}

	sortSlice(rows)

	//Add the column name
	var row []string
	row = append(row, "svc_name", "svc_namespace")
	for i := 1; i <= maxReplicasCount; i++ {
		row = append(row, "replica_"+strconv.Itoa(i)+"_ready")
	}
	rows = append([][]string{row}, rows...)

	// generate CSV, HTML and JSON outputs from rows and loadFromZeroResult
	err = GenerateOutput(inputs.Output, LoadOutputFilename, true, true, true, rows, loadFromZeroResult)
	if err != nil {
		fmt.Printf("failed to generate output: %s\n", err)
		return err
	}

	return nil
}

func loadAndMeasure(ctx context.Context, params *pkg.PerfParams, inputs pkg.LoadArgs, nsNameList []string, servicesListFunc func(context.Context, servingv1client.ServingV1Interface, []string, string, string, string) ([]ServicesToScale, error)) (pkg.LoadResult, error) {
	result := pkg.LoadResult{}
	ksvcClient, err := params.NewServingClient()
	if err != nil {
		return result, err
	}
	objs, err := servicesListFunc(ctx, ksvcClient, nsNameList, inputs.SvcPrefix, inputs.SvcRange, inputs.Svc)
	if err != nil {
		return result, err
	}
	count := len(objs)

	var wg sync.WaitGroup
	var m sync.Mutex
	wg.Add(count)
	for i := 0; i < count; i++ {
		go func(ndx int, m *sync.Mutex) {
			defer wg.Done()
			loadToolOutput, loadResult, err := runLoadFromZero(ctx, params, inputs, objs[ndx].Namespace, objs[ndx].Service)
			if err == nil {
				// print result(load test tool output, replicas result, pods result)
				if inputs.Verbose {
					fmt.Printf("\n[Verbose] Namespace %s, Service %s:\n", loadResult.ServiceNamespace, loadResult.ServiceName)
					fmt.Printf("\n[Verbose] Load tool(%s) output:\n%s\n", inputs.LoadTool, loadToolOutput)
					fmt.Printf("[Verbose] Deployment replicas changed from 0 to %d:\n", len(loadResult.ReplicaResults))
					fmt.Printf("replicas\tready_duration(seconds)\n")
					for i := 0; i < len(loadResult.ReplicaResults); i++ {
						fmt.Printf("%8d\t%23.3f\n", i, loadResult.ReplicaResults[i].ReplicaReadyDuration)
					}
					fmt.Printf("\n[Verbose] Pods changed from 0 to %d:\n", len(loadResult.PodResults))
					fmt.Printf("pods\tready_duration(seconds)\n")
					for i := 0; i < len(loadResult.PodResults); i++ {
						fmt.Printf("%4d\t%23.1f\n", i, loadResult.PodResults[i].PodReadyDuration)
					}
					fmt.Printf("\n---------------------------------------------------------------------------------\n")
				}
				m.Lock()
				result.Measurment = append(result.Measurment, loadResult)
				m.Unlock()
			} else {
				fmt.Printf("failed in runLoadFromZero: %s\n", err)
			}
		}(i, &m)
	}
	wg.Wait()

	return result, nil
}

func runLoadFromZero(ctx context.Context, params *pkg.PerfParams, inputs pkg.LoadArgs, namespace string, svc *servingv1.Service) (
	string, pkg.LoadFromZeroResult, error) {
	selector := labels.SelectorFromSet(labels.Set{
		serving.ServiceLabelKey: svc.Name,
	})
	var loadOutput string
	var loadResult pkg.LoadFromZeroResult
	var replicaResults []pkg.LoadReplicaResult
	var podResults []pkg.LoadPodResult

	watcher, err := params.ClientSet.AppsV1().Deployments(namespace).Watch(
		context.Background(), metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		m := fmt.Sprintf("unable to watch the deployment for the service: %v", err)
		log.Println(m)
		return "", loadResult, errors.New(m)
	}
	defer watcher.Stop()

	rdch := watcher.ResultChan() // replica duration channel
	pdch := make(chan struct{})  // pod duration channel
	errch := make(chan error, 1)

	endpoint, err := resolveEndpoint(ctx, params, inputs.ResolvableDomain, inputs.Https, svc)
	if err != nil {
		return "", loadResult, fmt.Errorf("failed to get the cluster endpoint: %w", err)
	}
	host := svc.Status.RouteStatusFields.URL.URL().Host

	loadStart := time.Now()
	log.Printf("Namespace %s, Service %s, load start\n", namespace, svc.Name)

	go func() {
		if inputs.LoadTool == "default" {
			loadOutput, err = runInternalVegeta(inputs, endpoint, host)
			if err != nil {
				errch <- fmt.Errorf("failed to run internal load tool: %w", err)
				return
			}
		} else {
			loadOutput, err = runExternalLoadTool(inputs, namespace, svc.Name, endpoint, host)
			if err != nil {
				errch <- fmt.Errorf("failed to run external load tool: %w", err)
				return
			}
		}
		loadEnd := time.Now()
		loadDuration := loadEnd.Sub(loadStart)
		log.Printf("Namespace %s, Service %s, load end, take off %.3f seconds\n", namespace, svc.Name, loadDuration.Seconds())

		time.Sleep(inputs.WaitPodsReadyDuration) //wait for all pods ready
		pdch <- struct{}{}
	}()

	for {
		select {
		case event := <-rdch:
			replicaResults = getReplicaResult(replicaResults, event, loadStart)
		case <-pdch:
			podResults, err = getPodResults(ctx, params, namespace, svc)
			if err != nil {
				return loadOutput, loadResult, err
			}
			// set loadResult
			loadResult = setLoadFromZeroResult(namespace, svc, replicaResults, podResults)
			return loadOutput, loadResult, nil
		case err := <-errch:
			return "", loadResult, err
		}
	}
}

// getSvcPod gets pod list by namespace and service name.
func getSvcPods(ctx context.Context, params *pkg.PerfParams, namespace string, svcName string) (PodList []corev1.Pod, err error) {
	selector := labels.SelectorFromSet(labels.Set{
		serving.ServiceLabelKey: svcName,
	})

	ksvcPodList, err := params.ClientSet.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return nil, err
	}

	return ksvcPodList.Items, nil
}

// getReplicasCount gets the maximum count of replicas in each service and the maximum count of replicas in all services
func getReplicasCount(loadResult pkg.LoadResult) (int, []int) {
	var maxReplicasCount int     // the maximum count in replicasCountList
	replicasCountList := []int{} // the maximum replicas count in each service
	for _, m := range loadResult.Measurment {
		count := 0

		for _, d := range m.ReplicaResults {
			if d.ReadyReplicasCount > count {
				count = d.ReadyReplicasCount
			}
		}
		replicasCountList = append(replicasCountList, count)
		if count > maxReplicasCount {
			maxReplicasCount = count
		}
	}
	return maxReplicasCount, replicasCountList
}

// runInternalVegeta runs internal load test tool(vegeta) using library, returns load output and error
func runInternalVegeta(inputs pkg.LoadArgs, endpoint string, host string) (output string, err error) {
	concurrency, err := strconv.ParseUint(inputs.LoadConcurrency, 10, 64)
	if err != nil {
		return "", fmt.Errorf("failed to get load concurrency: %s", err)
	}

	duration, err := time.ParseDuration(inputs.LoadDuration)
	if err != nil {
		return "", fmt.Errorf("failed to get load duration: %s", err)
	}

	rate := vegeta.Rate{Freq: 8, Per: time.Second}
	targeter := vegeta.NewStaticTargeter(vegeta.Target{
		Method: "GET",
		URL:    endpoint,
		Header: http.Header{
			"Host": []string{
				host,
			},
		},
	})

	attacker := vegeta.NewAttacker(vegeta.Workers(concurrency))

	var metrics vegeta.Metrics
	for res := range attacker.Attack(targeter, rate, duration, "Big Bang!") {
		metrics.Add(res)
	}
	defer metrics.Close()

	repText := vegeta.NewTextReporter(&metrics)

	buf := new(bytes.Buffer)
	err = repText.Report(buf)
	if err != nil {
		return "", fmt.Errorf("failed to write result to buffer: %s", err)
	}

	return buf.String(), nil
}

// runExternalLoadTool runs external load test tool(wrk or hey) using command line, returns load output and error
func runExternalLoadTool(inputs pkg.LoadArgs, namespace string, svcName string, endpoint string, host string) (output string, err error) {
	// Prepare command for load test tool
	cmd, wrkLua, err := loadCmdBuilder(inputs, namespace, svcName, endpoint, host)
	if err != nil {
		return "", fmt.Errorf("failed to run loadCmdBuilder: %s", err)
	}
	defer func() {
		// Delete wrk lua script
		if strings.EqualFold(inputs.LoadTool, "wrk") {
			err := deleteFile(wrkLua)
			if err != nil {
				fmt.Printf("%s\n", err)
			}
		}
	}()

	runCmd := exec.Command("/bin/sh", "-c", cmd)
	var loadOutput []byte
	loadOutput, err = runCmd.Output()
	output = string(loadOutput)
	if err != nil {
		return "", fmt.Errorf("failed to run load command: %s", err)
	}
	return output, nil
}

// loadCmdBuilder builds the command to run load tool, returns command and wrk lua script.
func loadCmdBuilder(inputs pkg.LoadArgs, namespace string, svcName string, endpoint string, host string) (string, string, error) {
	var cmd strings.Builder
	var wrkLuaFilename string
	if strings.EqualFold(inputs.LoadTool, "hey") {
		cmd.WriteString("hey")
		cmd.WriteString(" -c ")
		cmd.WriteString(inputs.LoadConcurrency)
		cmd.WriteString(" -z ")
		cmd.WriteString(inputs.LoadDuration)
		cmd.WriteString(" -host ")
		cmd.WriteString(host)
		cmd.WriteString(" ")
		cmd.WriteString(endpoint)
		return cmd.String(), "", nil
	}

	if strings.EqualFold(inputs.LoadTool, "wrk") {
		// creat lua script to config host of URL
		wrkLuaFilename = "./wrk_" + namespace + "_" + svcName + ".lua"
		var content strings.Builder
		content.WriteString("wrk.host = \"")
		content.WriteString(host)
		content.WriteString("\"")
		data := []byte(content.String())
		err := ioutil.WriteFile(wrkLuaFilename, data, 0644)
		if err != nil {
			return "", "", fmt.Errorf("write wrk lua script error: %w", err)
		}
		cmd.WriteString("wrk")
		cmd.WriteString(" -c ")
		cmd.WriteString(inputs.LoadConcurrency)
		cmd.WriteString(" -d ")
		cmd.WriteString(inputs.LoadDuration)
		cmd.WriteString(" -s ")
		cmd.WriteString(wrkLuaFilename)
		cmd.WriteString(" ")
		cmd.WriteString(endpoint)
		cmd.WriteString(" --latency")
		return cmd.String(), wrkLuaFilename, nil
	}

	return "", "", fmt.Errorf("kperf only support hey and wrk now")
}

// getReplicaResult get replicaResult by watching deployment, and append replicaResult to replicaResults
func getReplicaResult(replicaResults []pkg.LoadReplicaResult, event watch.Event, loadStart time.Time) []pkg.LoadReplicaResult {
	var replicaResult pkg.LoadReplicaResult
	dm := event.Object.(*v1.Deployment)
	readyReplicas := int(dm.Status.ReadyReplicas)
	if event.Type == watch.Modified && readyReplicas > len(replicaResults) {
		replicaResult.ReplicaReadyTime = time.Now()
		replicaResult.ReadyReplicasCount = readyReplicas
		replicaResult.ReplicaReadyDuration = replicaResult.ReplicaReadyTime.Sub(loadStart).Seconds()
		replicaResults = append(replicaResults, replicaResult)
	}
	return replicaResults
}

// getPodResults gets podReadyDuration of all pods and append result to podResults
func getPodResults(ctx context.Context, params *pkg.PerfParams, namespace string, svc *servingv1.Service) ([]pkg.LoadPodResult, error) {
	var r pkg.LoadPodResult
	var podResults []pkg.LoadPodResult
	podList, err := getSvcPods(ctx, params, namespace, svc.Name)
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(podList); i++ {
		pod := podList[i]
		podCreatedTime := pod.GetCreationTimestamp().Rfc3339Copy()
		present, PodReadyCondition := getPodCondition(&pod.Status, corev1.PodReady)
		if present == -1 {
			log.Println("failed to find Pod Condition PodReady and skip measuring")
			continue
		}
		podReadyTime := PodReadyCondition.LastTransitionTime.Rfc3339Copy()
		podReadyDuration := podReadyTime.Sub(podCreatedTime.Time).Seconds()
		r.PodCreateTime = podCreatedTime
		r.PodReadyTime = podReadyTime
		r.PodReadyDuration = podReadyDuration
		podResults = append(podResults, r)
	}
	return podResults, nil
}

// setLoadFromZeroResult sets every item of LoadFromZeroResult
func setLoadFromZeroResult(namespace string, svc *servingv1.Service, replicaResults []pkg.LoadReplicaResult, podResults []pkg.LoadPodResult) pkg.LoadFromZeroResult {
	var loadResult pkg.LoadFromZeroResult
	var totalReadyReplicas int
	for _, value := range replicaResults {
		if value.ReadyReplicasCount > totalReadyReplicas {
			totalReadyReplicas = value.ReadyReplicasCount
		}
	}
	loadResult.TotalReadyReplicas = totalReadyReplicas
	loadResult.TotalReadyPods = len(podResults)
	loadResult.ReplicaResults = replicaResults
	loadResult.PodResults = podResults

	loadResult.ServiceName = svc.Name
	loadResult.ServiceNamespace = namespace

	return loadResult
}
