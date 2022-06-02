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
	"path/filepath"

	"knative.dev/kperf/pkg/command/utils"

	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"

	_ "k8s.io/client-go/plugin/pkg/client/auth"

	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"k8s.io/apimachinery/pkg/watch"

	"knative.dev/kperf/pkg"
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
		Short: "Load and Measure Knative service",
		Long: `Scale Knative service from zero using load test tool and measure latency for service to scale form 0 to N

For example:
# To measure a Knative Service scaling from zero to N
kperf service load --namespace ktest --svc-prefix ktest --range 0,3 --load-tool wrk --load-duration 60s --load-concurrency 40 --verbose --output /tmp`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Flags().NFlag() == 0 {
				return fmt.Errorf("'service load' requires flag(s)")
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
	serviceLoadCommand.Flags().StringVarP(&loadArgs.SvcPrefix, "svc-prefix", "", "", "Service name prefix")
	serviceLoadCommand.Flags().StringVarP(&loadArgs.SvcRange, "range", "r", "", "Desired service range")
	serviceLoadCommand.Flags().BoolVarP(&loadArgs.Verbose, "verbose", "v", false, "Service verbose result")
	serviceLoadCommand.Flags().BoolVarP(&loadArgs.ResolvableDomain, "resolvable", "", false, "If Service endpoint resolvable url")
	serviceLoadCommand.Flags().DurationVarP(&loadArgs.WaitPodsReadyDuration, "wait-time", "w", 10*time.Second, "Time to wait for all pods to be ready")
	serviceLoadCommand.Flags().StringVarP(&loadArgs.LoadTool, "load-tool", "t", "wrk", "Select the load test tool, support wrk and hey")
	serviceLoadCommand.Flags().StringVarP(&loadArgs.LoadConcurrency, "load-concurrency", "c", "30", "total number of workers to run concurrently for the load test tool")
	serviceLoadCommand.Flags().StringVarP(&loadArgs.LoadDuration, "load-duration", "d", "60s", "Duration of the test for the load test tool")
	serviceLoadCommand.Flags().StringVarP(&loadArgs.Output, "output", "o", ".", "Measure result location")

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

	current := time.Now()
	outputLocation, err := utils.CheckOutputLocation(inputs.Output)
	if err != nil {
		fmt.Printf("failed to check measure output location: %s\n", err)
	}

	csvPath := filepath.Join(outputLocation, fmt.Sprintf("%s_%s.csv", current.Format(DateFormatString), LoadOutputFilename))
	err = utils.GenerateCSVFile(csvPath, rows)
	if err != nil {
		fmt.Printf("failed to generate CSV file and skip %s\n", err)
	}
	fmt.Printf("Measurement saved in CSV file %s\n", csvPath)

	jsonPath := filepath.Join(outputLocation, fmt.Sprintf("%s_%s.json", current.Format(DateFormatString), LoadOutputFilename))
	jsonData, err := json.Marshal(loadFromZeroResult)
	if err != nil {
		fmt.Printf("failed to generate json data and skip %s\n", err)
	}
	err = utils.GenerateJSONFile(jsonData, jsonPath)
	if err != nil {
		fmt.Printf("failed to generate json file and skip %s\n", err)
	}
	fmt.Printf("Measurement saved in JSON file %s\n", jsonPath)

	htmlPath := filepath.Join(outputLocation, fmt.Sprintf("%s_%s.html", current.Format(DateFormatString), LoadOutputFilename))
	err = utils.GenerateHTMLFile(csvPath, htmlPath)
	if err != nil {
		fmt.Printf("failed to generate HTML file and skip %s\n", err)
	}
	fmt.Printf("Visualized measurement saved in HTML file %s\n", htmlPath)

	return nil
}

func loadAndMeasure(ctx context.Context, params *pkg.PerfParams, inputs pkg.LoadArgs, nsNameList []string, servicesListFunc func(context.Context, servingv1client.ServingV1Interface, []string, string) []ServicesToScale) (pkg.LoadResult, error) {
	result := pkg.LoadResult{}
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
			loadToolOutput, loadResult, err := runLoadFromZero(ctx, params, inputs, objs[ndx].Namespace, objs[ndx].Service)
			if err == nil {
				// print result(load test tool output, replicas result, pods result)
				if inputs.Verbose {
					fmt.Printf("\n[Verbose] Namespace %s, Service %s:\n", loadResult.ServiceNamespace, loadResult.ServiceName)
					fmt.Printf("\n[Verbose] %s output:\n%s\n", inputs.LoadTool, loadToolOutput)
					fmt.Printf("[Verbose] Deployment replicas changed from 0 to %d:\n", len(loadResult.ReplicaResults))
					fmt.Printf("replicas\tready_duration(seconds)\n")
					for i := 0; i < len(loadResult.ReplicaResults); i++ {
						fmt.Printf("%13d\t%23.3f\n", i, loadResult.ReplicaResults[i].ReplicaReadyDuration)
					}
					fmt.Printf("\n[Verbose] Pods changed from 0 to %d:\n", len(loadResult.PodResults))
					fmt.Printf("pods\tready_duration(seconds)\n")
					for i := 0; i < len(loadResult.PodResults); i++ {
						fmt.Printf("%9d\t%23.1f\n", i, loadResult.PodResults[i].PodReadyDuration)
					}
					fmt.Printf("\n---------------------------------------------------------------------------------\n")
				}
				m.Lock()
				result.Measurment = append(result.Measurment, loadResult)
				m.Unlock()
			} else {
				fmt.Printf("result of load is error: %s", err)
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
	loadResult.ServiceName = svc.Name
	loadResult.ServiceNamespace = namespace

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
	errch := make(chan error)

	endpoint, err := resolveEndpoint(ctx, params, inputs.ResolvableDomain, svc)
	if err != nil {
		return "", loadResult, fmt.Errorf("failed to get the cluster endpoint: %w", err)
	}

	// Prepare for load test
	cmd, wrkLua, err := loadCmdBuilder(inputs, endpoint, namespace, svc)
	if err != nil {
		return "", loadResult, err
	}

	defer func() {
		// Delete wrk lua script
		if strings.EqualFold(inputs.LoadTool, "wrk") {
			_, fileError := os.Stat(wrkLua)
			if fileError == nil {
				removeError := os.Remove(wrkLua)
				if removeError != nil {
					log.Printf("remove %s error : %s", wrkLua, removeError)
				}
			}
		}
	}()

	loadStart := time.Now()
	m := fmt.Sprintf("Namespace %s, Service %s, load start", namespace, svc.Name)
	log.Println(m)

	go func() {
		runCmd := exec.Command("/bin/sh", "-c", cmd)
		var output []byte
		output, err = runCmd.Output()
		loadOutput = string(output)

		if err != nil {
			m := fmt.Sprintf("run load command error: %s", err)
			fmt.Println(m)
			return
		}

		loadEnd := time.Now()
		loadDuration := loadEnd.Sub(loadStart)
		m := fmt.Sprintf("Namespace %s, Service %s, load end, take off %.3f seconds", namespace, svc.Name, loadDuration.Seconds())
		log.Println(m)

		time.Sleep(inputs.WaitPodsReadyDuration * time.Second) //wait for all pods ready
		pdch <- struct{}{}
	}()

	var preReadyReplicas int
	for {
		select {
		case event := <-rdch:
			// get replicas ready duration by watching deployment event
			var r pkg.LoadReplicaResult
			dm := event.Object.(*v1.Deployment)
			readyReplicas := int(dm.Status.ReadyReplicas)
			if event.Type == watch.Modified && readyReplicas > preReadyReplicas {
				r.ReplicaReadyTime = time.Now()
				r.ReadyReplicasCount = readyReplicas
				r.ReplicaReadyDuration = r.ReplicaReadyTime.Sub(loadStart).Seconds()
				replicaResults = append(replicaResults, r)
				preReadyReplicas = readyReplicas
			}
		case <-pdch:
			// get pods ready duration by pod conditions
			var r pkg.LoadPodResult
			podList, err := getSvcPods(ctx, params, namespace, svc.Name)
			if err != nil {
				return "", loadResult, err
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
			return loadOutput, loadResult, nil
		case err := <-errch:
			return loadOutput, loadResult, err
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
	var replicasCountList []int // the maximum replicas count in each service
	var maxReplicasCount int    // the maximum count in replicasCountList
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

// loadCmdBuilder builds the command to run load tool, returns command and wrk lua script.
func loadCmdBuilder(inputs pkg.LoadArgs, endpoint string, namespace string, svc *servingv1.Service) (string, string, error) {
	var cmd strings.Builder
	var wrkLuaFilename string
	if strings.EqualFold(inputs.LoadTool, "hey") {
		cmd.WriteString("hey")
		cmd.WriteString(" -c ")
		cmd.WriteString(inputs.LoadConcurrency)
		cmd.WriteString(" -z ")
		cmd.WriteString(inputs.LoadDuration)
		cmd.WriteString(" -host ")
		cmd.WriteString(svc.Status.RouteStatusFields.URL.URL().Host)
		cmd.WriteString(" ")
		cmd.WriteString(endpoint)
		return cmd.String(), "", nil
	}

	if strings.EqualFold(inputs.LoadTool, "wrk") {
		// creat lua script to config host of URL
		wrkLuaFilename = "./wrk_" + namespace + "_" + svc.Name + ".lua"
		var content strings.Builder
		content.WriteString("wrk.host = \"")
		content.WriteString(svc.Status.RouteStatusFields.URL.URL().Host)
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
