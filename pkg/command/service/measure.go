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

package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/kperf/pkg/command/utils"

	"encoding/json"
	"path/filepath"

	"knative.dev/kperf/pkg"

	"github.com/montanaflynn/stats"
	"github.com/spf13/cobra"

	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"

	corev1 "k8s.io/api/core/v1"
	networkingv1api "knative.dev/networking/pkg/apis/networking/v1alpha1"
	autoscalingv1api "knative.dev/serving/pkg/apis/autoscaling/v1alpha1"
	servingv1api "knative.dev/serving/pkg/apis/serving/v1"
	v1 "knative.dev/serving/pkg/apis/serving/v1"
)

const (
	DateFormatString = "20060102150405"
)

func NewServiceMeasureCommand(p *pkg.PerfParams) *cobra.Command {
	measureArgs := measureArgs{}
	measureFinalResult := measureResult{}
	serviceMeasureCommand := &cobra.Command{
		Use:   "measure",
		Short: "Measure Knative service",
		Long: `Measure Knative service creation time

For example:
# To measure a Knative Service creation time running currently with 20 concurent jobs
kperf service measure --svc-perfix svc --range 1,200 --namespace ns --concurrency 20
`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Flags().NFlag() == 0 {
				return fmt.Errorf("'service measure' requires flag(s)")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			var lock sync.Mutex
			svcNamespacedName := make([][]string, 0)
			if cmd.Flags().Changed("namespace") {
				r := strings.Split(measureArgs.svcRange, ",")
				if len(r) != 2 {
					return fmt.Errorf("expected range like 1,500, given %s\n", measureArgs.svcRange)
				}

				start, err := strconv.Atoi(r[0])
				if err != nil {
					return err
				}
				end, err := strconv.Atoi(r[1])
				if err != nil {
					return err
				}

				for i := start; i <= end; i++ {
					sName := fmt.Sprintf("%s-%s", measureArgs.svcPrefix, strconv.Itoa(i))
					svcNamespacedName = append(svcNamespacedName, []string{sName, measureArgs.namespace})
				}
			}

			autoscalingClient, err := p.NewAutoscalingClient()
			if err != nil {
				return fmt.Errorf("failed to create autoscaling client%s\n", err)
			}
			servingClient, err := p.NewServingClient()
			if err != nil {
				return fmt.Errorf("failed to create serving client%s\n", err)
			}

			if cmd.Flags().Changed("namespace-range") && cmd.Flags().Changed("namespace-prefix") {
				r := strings.Split(measureArgs.namespaceRange, ",")
				if len(r) != 2 {
					return fmt.Errorf("expected namespace-range like 1,500, given %s\n", measureArgs.namespaceRange)
				}

				start, err := strconv.Atoi(r[0])
				if err != nil {
					return err
				}
				end, err := strconv.Atoi(r[1])
				if err != nil {
					return err
				}
				for i := start; i <= end; i++ {
					svcNsName := fmt.Sprintf("%s-%s", measureArgs.namespacePrefix, strconv.Itoa(i))
					svcList, err := servingClient.Services(svcNsName).List(context.TODO(), metav1.ListOptions{})
					if err != nil {
						return fmt.Errorf("failed to list service under namespace %s error:%v", svcNsName, err)
					}

					if len(svcList.Items) > 0 {
						for _, svc := range svcList.Items {
							if strings.HasPrefix(svc.Name, measureArgs.svcPrefix) {
								svcNamespacedName = append(svcNamespacedName, []string{svc.Name, svcNsName})
							}
						}
					} else {
						fmt.Printf("no service found under namespace %s and skip\n", svcNsName)
					}
				}
			}

			var verbose = cmd.Flags().Changed("verbose")
			if verbose {
				fmt.Printf("[Verbose] Start to measure Knative services...\n")
				fmt.Printf("[Verbose] Concurrency: %d\n", measureArgs.concurrency)
			}

			rows := make([][]string, 0)
			rawRows := make([][]string, 0)

			nwclient, err := p.NewNetworkingClient()
			if err != nil {
				return fmt.Errorf("failed to create networking client%s\n", err)
			}

			svcChannel := make(chan []string)
			group := sync.WaitGroup{}
			workerMeasureResults := make([]measureResult, measureArgs.concurrency)
			for i := 0; i < measureArgs.concurrency; i++ {
				workerMeasureResults[i] = measureResult{
					SvcReadyTime: make([]float64, 0),
				}
			}
			for i := 0; i < measureArgs.concurrency; i++ {
				go func(index int) {
					var (
						svcConfigurationsReadyDuration, svcReadyDuration, svcRoutesReadyDuration, podScheduledDuration,
						containersReadyDuration, queueProxyStartedDuration, userContrainerStartedDuration time.Duration
					)
					currentMeasureResult := workerMeasureResults[index]
					for j := range svcChannel {
						if len(j) != 2 {
							fmt.Printf("lack of service name or service namespace and skip")
							currentMeasureResult.Service.FailCount++
							workerMeasureResults[index] = currentMeasureResult
							group.Done()
							continue
						}
						svc := j[0]
						svcNs := j[1]
						svcIns, err := servingClient.Services(svcNs).Get(context.TODO(), svc, metav1.GetOptions{})
						if err != nil {
							fmt.Printf("failed to get Knative Service %s\n", err)
							if strings.Contains(err.Error(), "not found") {
								currentMeasureResult.Service.NotFoundCount++
								workerMeasureResults[index] = currentMeasureResult
								group.Done()
								continue
							} else {
								currentMeasureResult.Service.FailCount++
								workerMeasureResults[index] = currentMeasureResult
								group.Done()
								continue
							}
						}
						if !svcIns.IsReady() {
							fmt.Printf("service %s/%s not ready and skip measuring\n", svc, svcNs)
							currentMeasureResult.Service.NotReadyCount++
							workerMeasureResults[index] = currentMeasureResult
							group.Done()
							continue
						}

						svcCreatedTime := svcIns.GetCreationTimestamp().Rfc3339Copy()
						svcConfigurationsReady := svcIns.Status.GetCondition(servingv1api.ServiceConditionConfigurationsReady).LastTransitionTime.Inner.Rfc3339Copy()
						svcRoutesReady := svcIns.Status.GetCondition(servingv1api.ServiceConditionRoutesReady).LastTransitionTime.Inner.Rfc3339Copy()

						svcConfigurationsReadyDuration = svcConfigurationsReady.Sub(svcCreatedTime.Time)
						svcRoutesReadyDuration = svcRoutesReady.Sub(svcCreatedTime.Time)
						svcReadyDuration = svcRoutesReady.Sub(svcCreatedTime.Time)

						cfgIns, err := servingClient.Configurations(svcNs).Get(context.TODO(), svc, metav1.GetOptions{})
						if err != nil {
							fmt.Printf("failed to get Configuration and skip measuring %s\n", err)
							currentMeasureResult.Service.NotReadyCount++
							workerMeasureResults[index] = currentMeasureResult
							group.Done()
							continue
						}
						revisionName := cfgIns.Status.LatestReadyRevisionName

						revisionIns, err := servingClient.Revisions(svcNs).Get(context.TODO(), revisionName, metav1.GetOptions{})
						if err != nil {
							fmt.Printf("failed to get Revision and skip measuring %s\n", err)
							currentMeasureResult.Service.NotReadyCount++
							workerMeasureResults[index] = currentMeasureResult
							group.Done()
							continue
						}

						revisionCreatedTime := revisionIns.GetCreationTimestamp().Rfc3339Copy()
						revisionReadyTime := revisionIns.Status.GetCondition(v1.RevisionConditionReady).LastTransitionTime.Inner.Rfc3339Copy()
						revisionReadyDuration := revisionReadyTime.Sub(revisionCreatedTime.Time)

						label := fmt.Sprintf("serving.knative.dev/revision=%s", revisionName)
						podList, err := p.ClientSet.CoreV1().Pods(svcNs).List(context.TODO(), metav1.ListOptions{LabelSelector: label})
						if err != nil {
							fmt.Printf("list Pods of revision[%s] error :%v", revisionName, err)
							currentMeasureResult.Service.NotReadyCount++
							workerMeasureResults[index] = currentMeasureResult
							group.Done()
							continue
						}

						deploymentName := revisionName + "-deployment"
						deploymentIns, err := p.ClientSet.AppsV1().Deployments(svcNs).Get(context.TODO(), deploymentName, metav1.GetOptions{})
						if err != nil {
							fmt.Printf("failed to find deployment of revision[%s] error:%v", revisionName, err)
							currentMeasureResult.Service.NotReadyCount++
							workerMeasureResults[index] = currentMeasureResult
							group.Done()
							continue
						}

						deploymentCreatedTime := deploymentIns.GetCreationTimestamp().Rfc3339Copy()
						deploymentCreatedDuration := deploymentCreatedTime.Sub(revisionCreatedTime.Time)

						var podCreatedTime, podScheduledTime, containersReadyTime, queueProxyStartedTime,
							userContrainerStartedTime metav1.Time
						if len(podList.Items) > 0 {
							pod := podList.Items[0]
							podCreatedTime = pod.GetCreationTimestamp().Rfc3339Copy()
							present, PodScheduledCdt := getPodCondition(&pod.Status, corev1.PodScheduled)
							if present == -1 {
								fmt.Printf("failed to find Pod Condition PodScheduled and skip measuring")
								currentMeasureResult.Service.NotReadyCount++
								workerMeasureResults[index] = currentMeasureResult
								group.Done()
								continue
							}
							podScheduledTime = PodScheduledCdt.LastTransitionTime.Rfc3339Copy()
							present, containersReadyCdt := getPodCondition(&pod.Status, corev1.ContainersReady)
							if present == -1 {
								fmt.Printf("failed to find Pod Condition ContainersReady and skip measuring")
								currentMeasureResult.Service.NotReadyCount++
								workerMeasureResults[index] = currentMeasureResult
								group.Done()
								continue
							}
							containersReadyTime = containersReadyCdt.LastTransitionTime.Rfc3339Copy()
							podScheduledDuration = podScheduledTime.Sub(podCreatedTime.Time)
							containersReadyDuration = containersReadyTime.Sub(podCreatedTime.Time)

							queueProxyStatus, found := getContainerStatus(pod.Status.ContainerStatuses, "queue-proxy")
							if !found {
								fmt.Printf("failed to get queue-proxy container status and skip, error:%v", err)
								currentMeasureResult.Service.NotReadyCount++
								workerMeasureResults[index] = currentMeasureResult
								group.Done()
								continue
							}
							queueProxyStartedTime = queueProxyStatus.State.Running.StartedAt.Rfc3339Copy()

							userContrainerStatus, found := getContainerStatus(pod.Status.ContainerStatuses, "user-container")
							if !found {
								fmt.Printf("failed to get user-container container status and skip, error:%v", err)
								currentMeasureResult.Service.NotReadyCount++
								workerMeasureResults[index] = currentMeasureResult
								group.Done()
								continue
							}
							userContrainerStartedTime = userContrainerStatus.State.Running.StartedAt.Rfc3339Copy()

							queueProxyStartedDuration = queueProxyStartedTime.Sub(podCreatedTime.Time)
							userContrainerStartedDuration = userContrainerStartedTime.Sub(podCreatedTime.Time)
						}
						// TODO: Need to figure out a better way to measure PA time as its status keeps changing even after service creation.

						kpaIns, err := autoscalingClient.PodAutoscalers(svcNs).Get(context.TODO(), revisionName, metav1.GetOptions{})
						if err != nil {
							fmt.Printf("failed to get PodAutoscaler %s\n", err)
							currentMeasureResult.Service.NotReadyCount++
							workerMeasureResults[index] = currentMeasureResult
							group.Done()
							continue
						}
						kpaCreatedTime := kpaIns.GetCreationTimestamp().Rfc3339Copy()
						kpaActiveTime := kpaIns.Status.GetCondition(autoscalingv1api.PodAutoscalerConditionActive).LastTransitionTime.Inner.Rfc3339Copy()
						kpaActiveDuration := kpaActiveTime.Sub(kpaCreatedTime.Time)

						sksIns, err := nwclient.ServerlessServices(svcNs).Get(context.TODO(), revisionName, metav1.GetOptions{})
						if err != nil {
							fmt.Printf("failed to get ServerlessService %s\n", err)
							currentMeasureResult.Service.NotReadyCount++
							workerMeasureResults[index] = currentMeasureResult
							group.Done()
							continue
						}
						sksCreatedTime := sksIns.GetCreationTimestamp().Rfc3339Copy()
						sksActivatorEndpointsPopulatedTime := sksIns.Status.GetCondition(networkingv1api.ActivatorEndpointsPopulated).LastTransitionTime.Inner.Rfc3339Copy()
						sksEndpointsPopulatedTime := sksIns.Status.GetCondition(networkingv1api.ServerlessServiceConditionEndspointsPopulated).LastTransitionTime.Inner.Rfc3339Copy()
						sksReadyTime := sksIns.Status.GetCondition(networkingv1api.ServerlessServiceConditionReady).LastTransitionTime.Inner.Rfc3339Copy()
						sksActivatorEndpointsPopulatedDuration := sksActivatorEndpointsPopulatedTime.Sub(sksCreatedTime.Time)
						sksEndpointsPopulatedDuration := sksEndpointsPopulatedTime.Sub(sksCreatedTime.Time)
						sksReadyDuration := sksReadyTime.Sub(sksCreatedTime.Time)

						ingressIns, err := nwclient.Ingresses(svcNs).Get(context.TODO(), svc, metav1.GetOptions{})
						if err != nil {
							fmt.Printf("failed to get Ingress %s\n", err)
							currentMeasureResult.Service.NotReadyCount++
							workerMeasureResults[index] = currentMeasureResult
							group.Done()
							continue
						}
						ingressCreatedTime := ingressIns.GetCreationTimestamp().Rfc3339Copy()
						ingressNetworkConfiguredTime := ingressIns.Status.GetCondition(networkingv1api.IngressConditionNetworkConfigured).LastTransitionTime.Inner.Rfc3339Copy()
						ingressLoadBalancerReadyTime := ingressIns.Status.GetCondition(networkingv1api.IngressConditionLoadBalancerReady).LastTransitionTime.Inner.Rfc3339Copy()
						ingressNetworkConfiguredDuration := ingressNetworkConfiguredTime.Sub(ingressCreatedTime.Time)
						ingressLoadBalancerReadyDuration := ingressLoadBalancerReadyTime.Sub(ingressNetworkConfiguredTime.Time)
						ingressReadyDuration := ingressLoadBalancerReadyTime.Sub(ingressCreatedTime.Time)

						lock.Lock()
						currentMeasureResult.Service.ReadyCount++
						rows = append(rows, []string{svc, svcNs,
							fmt.Sprintf("%d", int(svcConfigurationsReadyDuration.Seconds())),
							fmt.Sprintf("%d", int(revisionReadyDuration.Seconds())),
							fmt.Sprintf("%d", int(deploymentCreatedDuration.Seconds())),
							fmt.Sprintf("%d", int(podScheduledDuration.Seconds())),
							fmt.Sprintf("%d", int(containersReadyDuration.Seconds())),
							fmt.Sprintf("%d", int(queueProxyStartedDuration.Seconds())),
							fmt.Sprintf("%d", int(userContrainerStartedDuration.Seconds())),
							fmt.Sprintf("%d", int(svcRoutesReadyDuration.Seconds())),
							fmt.Sprintf("%d", int(kpaActiveDuration.Seconds())),
							fmt.Sprintf("%d", int(sksReadyDuration.Seconds())),
							fmt.Sprintf("%d", int(sksActivatorEndpointsPopulatedDuration.Seconds())),
							fmt.Sprintf("%d", int(sksEndpointsPopulatedDuration.Seconds())),
							fmt.Sprintf("%d", int(ingressReadyDuration.Seconds())),
							fmt.Sprintf("%d", int(ingressNetworkConfiguredDuration.Seconds())),
							fmt.Sprintf("%d", int(ingressLoadBalancerReadyDuration.Seconds())),
							fmt.Sprintf("%d", int(svcReadyDuration.Seconds())),
						})

						rawRows = append(rawRows, []string{svc, svcNs,
							svcCreatedTime.String(),
							svcConfigurationsReady.Rfc3339Copy().String(),
							revisionIns.GetCreationTimestamp().Rfc3339Copy().String(),
							revisionReadyTime.String(),
							deploymentCreatedTime.String(),
							podCreatedTime.String(),
							podScheduledTime.String(),
							containersReadyTime.String(),
							queueProxyStartedTime.String(),
							userContrainerStartedTime.String(),
							svcRoutesReady.String(),
							kpaCreatedTime.String(),
							kpaActiveTime.String(),
							sksCreatedTime.String(),
							sksActivatorEndpointsPopulatedTime.String(),
							sksEndpointsPopulatedTime.String(),
							ingressCreatedTime.String(),
							ingressNetworkConfiguredTime.String(),
							ingressLoadBalancerReadyTime.String()})

						if verbose {
							fmt.Printf("[Verbose] Service %s: Service Configuration Ready Duration is %s/%fs\n",
								svc, svcConfigurationsReadyDuration, svcConfigurationsReadyDuration.Seconds())
							fmt.Printf("[Verbose] Service %s: - Service Revision Ready Duration is %s/%fs\n",
								svc, revisionReadyDuration, revisionReadyDuration.Seconds())
							fmt.Printf("[Verbose] Service %s:   - Service Deployment Created Duration is %s/%fs\n",
								svc, deploymentCreatedDuration, deploymentCreatedDuration.Seconds())
							fmt.Printf("[Verbose] Service %s:     - Service Pod Scheduled Duration is %s/%fs\n",
								svc, podScheduledDuration, podScheduledDuration.Seconds())
							fmt.Printf("[Verbose] Service %s:     - Service Pod Containers Ready Duration is %s/%fs\n",
								svc, containersReadyDuration, containersReadyDuration.Seconds())
							fmt.Printf("[Verbose] Service %s:       - Service Pod queue-proxy Started Duration is %s/%fs\n",
								svc, queueProxyStartedDuration, queueProxyStartedDuration.Seconds())
							fmt.Printf("[Verbose] Service %s:       - Service Pod user-container Started Duration is %s/%fs\n",
								svc, userContrainerStartedDuration, userContrainerStartedDuration.Seconds())
							fmt.Printf("[Verbose] Service %s:   - Service PodAutoscaler Active Duration is %s/%fs\n",
								svc, kpaActiveDuration, kpaActiveDuration.Seconds())
							fmt.Printf("[Verbose] Service %s:     - Service ServerlessService Ready Duration is %s/%fs\n",
								svc, sksReadyDuration, sksReadyDuration.Seconds())
							fmt.Printf("[Verbose] Service %s:       - Service ServerlessService ActivatorEndpointsPopulated Duration is %s/%fs\n",
								svc, sksActivatorEndpointsPopulatedDuration, sksActivatorEndpointsPopulatedDuration.Seconds())
							fmt.Printf("[Verbose] Service %s:       - Service ServerlessService EndpointsPopulated Duration is %s/%fs\n",
								svc, sksEndpointsPopulatedDuration, sksEndpointsPopulatedDuration.Seconds())
							fmt.Printf("[Verbose] Service %s: Service Route Ready Duration is %s/%fs\n", svc,
								svcRoutesReadyDuration, svcRoutesReadyDuration.Seconds())
							fmt.Printf("[Verbose] Service %s: - Service Ingress Ready Duration is %s/%fs\n",
								svc, ingressReadyDuration, ingressReadyDuration.Seconds())
							fmt.Printf("[Verbose] Service %s:   - Service Ingress Network Configured Duration is %s/%fs\n",
								svc, ingressNetworkConfiguredDuration, ingressNetworkConfiguredDuration.Seconds())
							fmt.Printf("[Verbose] Service %s:   - Service Ingress LoadBalancer Ready Duration is %s/%fs\n",
								svc, ingressLoadBalancerReadyDuration, ingressLoadBalancerReadyDuration.Seconds())
							fmt.Printf("[Verbose] Service %s: Overall Service Ready Duration is %s/%fs\n",
								svc, svcReadyDuration, svcReadyDuration.Seconds())
						}

						currentMeasureResult.Sums.svcConfigurationsReadySum += svcConfigurationsReadyDuration.Seconds()
						currentMeasureResult.Sums.revisionReadySum += revisionReadyDuration.Seconds()
						currentMeasureResult.Sums.deploymentCreatedSum += deploymentCreatedDuration.Seconds()
						currentMeasureResult.Sums.podScheduledSum += podScheduledDuration.Seconds()
						currentMeasureResult.Sums.containersReadySum += containersReadyDuration.Seconds()
						currentMeasureResult.Sums.queueProxyStartedSum += queueProxyStartedDuration.Seconds()
						currentMeasureResult.Sums.userContrainerStartedSum += userContrainerStartedDuration.Seconds()
						currentMeasureResult.Sums.svcRoutesReadySum += svcRoutesReadyDuration.Seconds()
						currentMeasureResult.Sums.kpaActiveSum += kpaActiveDuration.Seconds()
						currentMeasureResult.Sums.sksReadySum += sksReadyDuration.Seconds()
						currentMeasureResult.Sums.sksActivatorEndpointsPopulatedSum += sksActivatorEndpointsPopulatedDuration.Seconds()
						currentMeasureResult.Sums.sksEndpointsPopulatedSum += sksEndpointsPopulatedDuration.Seconds()
						currentMeasureResult.Sums.ingressReadySum += ingressReadyDuration.Seconds()
						currentMeasureResult.Sums.ingressNetworkConfiguredSum += ingressNetworkConfiguredDuration.Seconds()
						currentMeasureResult.Sums.ingressLoadBalancerReadySum += ingressLoadBalancerReadyDuration.Seconds()
						currentMeasureResult.Sums.svcReadySum += svcReadyDuration.Seconds()
						currentMeasureResult.SvcReadyTime = append(currentMeasureResult.SvcReadyTime, svcReadyDuration.Seconds())
						workerMeasureResults[index] = currentMeasureResult
						lock.Unlock()
						group.Done()
					}
				}(i)
			}
			if len(svcNamespacedName) == 0 {
				return errors.New("no service found to measure")
			}

			for _, item := range svcNamespacedName {
				group.Add(1)
				svcChannel <- item

			}

			group.Wait()

			for i := 0; i < measureArgs.concurrency; i++ {
				measureFinalResult.Sums.svcConfigurationsReadySum += workerMeasureResults[i].Sums.svcConfigurationsReadySum
				measureFinalResult.Sums.revisionReadySum += workerMeasureResults[i].Sums.revisionReadySum
				measureFinalResult.Sums.deploymentCreatedSum += workerMeasureResults[i].Sums.deploymentCreatedSum
				measureFinalResult.Sums.podScheduledSum += workerMeasureResults[i].Sums.podScheduledSum
				measureFinalResult.Sums.containersReadySum += workerMeasureResults[i].Sums.containersReadySum
				measureFinalResult.Sums.queueProxyStartedSum += workerMeasureResults[i].Sums.queueProxyStartedSum
				measureFinalResult.Sums.userContrainerStartedSum += workerMeasureResults[i].Sums.userContrainerStartedSum
				measureFinalResult.Sums.svcRoutesReadySum += workerMeasureResults[i].Sums.svcRoutesReadySum
				measureFinalResult.Sums.kpaActiveSum += workerMeasureResults[i].Sums.kpaActiveSum
				measureFinalResult.Sums.sksReadySum += workerMeasureResults[i].Sums.sksReadySum
				measureFinalResult.Sums.sksActivatorEndpointsPopulatedSum += workerMeasureResults[i].Sums.sksActivatorEndpointsPopulatedSum
				measureFinalResult.Sums.sksEndpointsPopulatedSum += workerMeasureResults[i].Sums.sksEndpointsPopulatedSum
				measureFinalResult.Sums.ingressReadySum += workerMeasureResults[i].Sums.ingressReadySum
				measureFinalResult.Sums.ingressNetworkConfiguredSum += workerMeasureResults[i].Sums.ingressNetworkConfiguredSum
				measureFinalResult.Sums.ingressLoadBalancerReadySum += workerMeasureResults[i].Sums.ingressLoadBalancerReadySum
				measureFinalResult.SvcReadyTime = append(measureFinalResult.SvcReadyTime, workerMeasureResults[i].SvcReadyTime...)
				measureFinalResult.Sums.svcReadySum += workerMeasureResults[i].Sums.svcReadySum
				measureFinalResult.Service.ReadyCount += workerMeasureResults[i].Service.ReadyCount
				measureFinalResult.Service.NotReadyCount += workerMeasureResults[i].Service.NotReadyCount
				measureFinalResult.Service.NotFoundCount += workerMeasureResults[i].Service.NotFoundCount
				measureFinalResult.Service.FailCount += workerMeasureResults[i].Service.FailCount
			}

			sortSlice(rows)
			sortSlice(rawRows)

			rows = append([][]string{{"svc_name", "svc_namespace", "configuration_ready", "revision_ready",
				"deployment_created", "pod_scheduled", "containers_ready", "queue-proxy_started", "user-container_started",
				"route_ready", "kpa_active", "sks_ready", "sks_activator_endpoints_populated", "sks_endpoints_populated",
				"ingress_ready", "ingress_config_ready", "ingress_lb_ready", "overall_ready"}}, rows...)

			rawRows = append([][]string{{"svc_name", "svc_namespace",
				"svc_created",
				"configuration_ready",
				"revision_created",
				"revision_ready",
				"deployment_created",
				"pod_created",
				"pod_scheduled",
				"containers_ready",
				"queue-proxy_started",
				"user-container_started",
				"route_ready",
				"kpa_created",
				"kpa_active",
				"sks_created",
				"sks_activator_endpoints_populated",
				"sks_endpoints_populated",
				"ingress_created",
				"ingress_config_ready",
				"ingress_lb_ready"}}, rawRows...)
			total := measureFinalResult.Service.ReadyCount + measureFinalResult.Service.NotReadyCount + measureFinalResult.Service.NotFoundCount + measureFinalResult.Service.FailCount

			knativeVersion := getKnativeVersion(p)
			ingressInfo := getIngressController(p)
			measureFinalResult.KnativeInfo.ServingVersion = knativeVersion["serving"]
			measureFinalResult.KnativeInfo.EventingVersion = knativeVersion["eventing"]
			measureFinalResult.KnativeInfo.IngressController = ingressInfo["ingressController"]
			measureFinalResult.KnativeInfo.IngressVersion = ingressInfo["version"]

			if measureFinalResult.Service.ReadyCount > 0 {
				fmt.Printf("-------- Measurement --------\n")
				fmt.Printf("Basic Information:\n")
				fmt.Printf("  - Knative Versions:\n")
				fmt.Printf("    Serving: %v\n", measureFinalResult.KnativeInfo.ServingVersion)
				fmt.Printf("    Eventing: %v\n", measureFinalResult.KnativeInfo.EventingVersion)
				fmt.Printf("  - Ingress Information:\n")
				fmt.Printf("    Controller: %v\n", measureFinalResult.KnativeInfo.IngressController)
				fmt.Printf("    Version: %v\n", measureFinalResult.KnativeInfo.IngressVersion)
				fmt.Printf("Total: %d | Ready: %d NotReady: %d NotFound: %d Fail: %d\n", total, measureFinalResult.Service.ReadyCount, measureFinalResult.Service.NotReadyCount, measureFinalResult.Service.NotFoundCount, measureFinalResult.Service.FailCount)
				fmt.Printf("Service Configuration Duration:\n")
				fmt.Printf("Total: %fs\n", measureFinalResult.Sums.svcConfigurationsReadySum)
				measureFinalResult.Result.AverageSvcConfigurationReadySum = measureFinalResult.Sums.svcConfigurationsReadySum / float64(measureFinalResult.Service.ReadyCount)
				fmt.Printf("Average: %fs\n", measureFinalResult.Result.AverageSvcConfigurationReadySum)

				fmt.Printf("- Service Revision Duration:\n")
				fmt.Printf("  Total: %fs\n", measureFinalResult.Sums.revisionReadySum)
				measureFinalResult.Result.AverageRevisionReadySum = measureFinalResult.Sums.revisionReadySum / float64(measureFinalResult.Service.ReadyCount)
				fmt.Printf("  Average: %fs\n", measureFinalResult.Result.AverageRevisionReadySum)

				fmt.Printf("  - Service Deployment Created Duration:\n")
				fmt.Printf("    Total: %fs\n", measureFinalResult.Sums.deploymentCreatedSum)
				measureFinalResult.Result.AverageDeploymentCreatedSum = measureFinalResult.Sums.deploymentCreatedSum / float64(measureFinalResult.Service.ReadyCount)
				fmt.Printf("    Average: %fs\n", measureFinalResult.Result.AverageDeploymentCreatedSum)

				fmt.Printf("    - Service Pod Scheduled Duration:\n")
				fmt.Printf("      Total: %fs\n", measureFinalResult.Sums.podScheduledSum)
				measureFinalResult.Result.AveragePodScheduledSum = measureFinalResult.Sums.podScheduledSum / float64(measureFinalResult.Service.ReadyCount)
				fmt.Printf("      Average: %fs\n", measureFinalResult.Result.AveragePodScheduledSum)

				fmt.Printf("    - Service Pod Containers Ready Duration:\n")
				fmt.Printf("      Total: %fs\n", measureFinalResult.Sums.containersReadySum)
				measureFinalResult.Result.AverageContainersReadySum = measureFinalResult.Sums.containersReadySum / float64(measureFinalResult.Service.ReadyCount)
				fmt.Printf("      Average: %fs\n", measureFinalResult.Result.AverageContainersReadySum)

				fmt.Printf("      - Service Pod queue-proxy Started Duration:\n")
				fmt.Printf("        Total: %fs\n", measureFinalResult.Sums.queueProxyStartedSum)
				measureFinalResult.Result.AverageQueueProxyStartedSum = measureFinalResult.Sums.queueProxyStartedSum / float64(measureFinalResult.Service.ReadyCount)
				fmt.Printf("        Average: %fs\n", measureFinalResult.Result.AverageQueueProxyStartedSum)

				fmt.Printf("      - Service Pod user-container Started Duration:\n")
				fmt.Printf("        Total: %fs\n", measureFinalResult.Sums.userContrainerStartedSum)
				measureFinalResult.Result.AverageUserContrainerStartedSum = measureFinalResult.Sums.userContrainerStartedSum / float64(measureFinalResult.Service.ReadyCount)
				fmt.Printf("        Average: %fs\n", measureFinalResult.Result.AverageUserContrainerStartedSum)

				fmt.Printf("  - Service PodAutoscaler Active Duration:\n")
				fmt.Printf("    Total: %fs\n", measureFinalResult.Sums.kpaActiveSum)
				measureFinalResult.Result.AverageKpaActiveSum = measureFinalResult.Sums.kpaActiveSum / float64(measureFinalResult.Service.ReadyCount)
				fmt.Printf("    Average: %fs\n", measureFinalResult.Result.AverageKpaActiveSum)

				fmt.Printf("    - Service ServerlessService Ready Duration:\n")
				fmt.Printf("      Total: %fs\n", measureFinalResult.Sums.sksReadySum)
				measureFinalResult.Result.AverageSksReadySum = measureFinalResult.Sums.sksReadySum / float64(measureFinalResult.Service.ReadyCount)
				fmt.Printf("      Average: %fs\n", measureFinalResult.Result.AverageSksReadySum)

				fmt.Printf("      - Service ServerlessService ActivatorEndpointsPopulated Duration:\n")
				fmt.Printf("        Total: %fs\n", measureFinalResult.Sums.sksActivatorEndpointsPopulatedSum)
				measureFinalResult.Result.AverageSksActivatorEndpointsPopulatedSum = measureFinalResult.Sums.sksActivatorEndpointsPopulatedSum / float64(measureFinalResult.Service.ReadyCount)
				fmt.Printf("        Average: %fs\n", measureFinalResult.Result.AverageSksActivatorEndpointsPopulatedSum)

				fmt.Printf("      - Service ServerlessService EndpointsPopulated Duration:\n")
				fmt.Printf("        Total: %fs\n", measureFinalResult.Sums.sksEndpointsPopulatedSum)
				measureFinalResult.Result.AverageSksEndpointsPopulatedSum = measureFinalResult.Sums.sksEndpointsPopulatedSum / float64(measureFinalResult.Service.ReadyCount)
				fmt.Printf("        Average: %fs\n", measureFinalResult.Result.AverageSksEndpointsPopulatedSum)

				fmt.Printf("\nService Route Ready Duration:\n")
				fmt.Printf("Total: %fs\n", measureFinalResult.Sums.svcRoutesReadySum)
				measureFinalResult.Result.AverageSvcRoutesReadySum = measureFinalResult.Sums.svcRoutesReadySum / float64(measureFinalResult.Service.ReadyCount)
				fmt.Printf("Average: %fs\n", measureFinalResult.Result.AverageSvcRoutesReadySum)

				fmt.Printf("- Service Ingress Ready Duration:\n")
				fmt.Printf("  Total: %fs\n", measureFinalResult.Sums.ingressReadySum)
				measureFinalResult.Result.AverageIngressReadySum = measureFinalResult.Sums.ingressReadySum / float64(measureFinalResult.Service.ReadyCount)
				fmt.Printf("  Average: %fs\n", measureFinalResult.Result.AverageIngressReadySum)

				fmt.Printf("  - Service Ingress Network Configured Duration:\n")
				fmt.Printf("    Total: %fs\n", measureFinalResult.Sums.ingressNetworkConfiguredSum)
				measureFinalResult.Result.AverageIngressNetworkConfiguredSum = measureFinalResult.Sums.ingressNetworkConfiguredSum / float64(measureFinalResult.Service.ReadyCount)
				fmt.Printf("    Average: %fs\n", measureFinalResult.Result.AverageIngressNetworkConfiguredSum)

				fmt.Printf("  - Service Ingress LoadBalancer Ready Duration:\n")
				fmt.Printf("    Total: %fs\n", measureFinalResult.Sums.ingressLoadBalancerReadySum)
				measureFinalResult.Result.AverageIngressLoadBalancerReadySum = measureFinalResult.Sums.ingressLoadBalancerReadySum / float64(measureFinalResult.Service.ReadyCount)
				fmt.Printf("    Average: %fs\n", measureFinalResult.Result.AverageIngressLoadBalancerReadySum)

				fmt.Printf("\n-----------------------------\n")
				fmt.Printf("Overall Service Ready Measurement:\n")
				fmt.Printf("Total: %d | Ready: %d (%.2f%s)  NotReady: %d (%.2f%s)  NotFound: %d (%.2f%s)  Fail: %d (%.2f%s) \n", total,
					measureFinalResult.Service.ReadyCount, float64(measureFinalResult.Service.ReadyCount)/float64(total)*100, "%",
					measureFinalResult.Service.NotReadyCount, float64(measureFinalResult.Service.NotReadyCount)/float64(total)*100, "%",
					measureFinalResult.Service.NotFoundCount, float64(measureFinalResult.Service.NotFoundCount)/float64(total)*100, "%",
					measureFinalResult.Service.FailCount, float64(measureFinalResult.Service.FailCount)/float64(total)*100, "%")
				measureFinalResult.Result.OverallTotal = measureFinalResult.Sums.svcReadySum
				fmt.Printf("Total: %fs\n", measureFinalResult.Result.OverallTotal)
				measureFinalResult.Result.OverallAverage = measureFinalResult.Sums.svcReadySum / float64(measureFinalResult.Service.ReadyCount)
				fmt.Printf("Average: %fs\n", measureFinalResult.Result.OverallAverage)

				measureFinalResult.Result.OverallMedian, _ = stats.Median(measureFinalResult.SvcReadyTime)
				fmt.Printf("Median: %fs\n", measureFinalResult.Result.OverallMedian)

				measureFinalResult.Result.OverallMin, _ = stats.Min(measureFinalResult.SvcReadyTime)
				fmt.Printf("Min: %fs\n", measureFinalResult.Result.OverallMin)

				measureFinalResult.Result.OverallMax, _ = stats.Max(measureFinalResult.SvcReadyTime)
				fmt.Printf("Max: %fs\n", measureFinalResult.Result.OverallMax)

				measureFinalResult.Result.P50, _ = stats.Percentile(measureFinalResult.SvcReadyTime, 50)
				fmt.Printf("Percentile50: %fs\n", measureFinalResult.Result.P50)

				measureFinalResult.Result.P90, _ = stats.Percentile(measureFinalResult.SvcReadyTime, 90)
				fmt.Printf("Percentile90: %fs\n", measureFinalResult.Result.P90)

				measureFinalResult.Result.P95, _ = stats.Percentile(measureFinalResult.SvcReadyTime, 95)
				fmt.Printf("Percentile95: %fs\n", measureFinalResult.Result.P95)

				measureFinalResult.Result.P98, _ = stats.Percentile(measureFinalResult.SvcReadyTime, 98)
				fmt.Printf("Percentile98: %fs\n", measureFinalResult.Result.P98)

				measureFinalResult.Result.P99, _ = stats.Percentile(measureFinalResult.SvcReadyTime, 99)
				fmt.Printf("Percentile99: %fs\n", measureFinalResult.Result.P99)

				current := time.Now()
				outputLocation, err := utils.CheckOutputLocation(measureArgs.output)
				if err != nil {
					fmt.Printf("failed to check measure output location: %s\n", err)
				}
				rawPath := filepath.Join(outputLocation, fmt.Sprintf("%s_%s", current.Format(DateFormatString), "raw_ksvc_creation_time.csv"))
				err = utils.GenerateCSVFile(rawPath, rawRows)
				if err != nil {
					fmt.Printf("failed to generate raw timestamp file and skip %s\n", err)
				}
				fmt.Printf("Raw Timestamp saved in CSV file %s\n", rawPath)

				csvPath := filepath.Join(outputLocation, fmt.Sprintf("%s_%s", current.Format(DateFormatString), "ksvc_creation_time.csv"))
				err = utils.GenerateCSVFile(csvPath, rows)
				if err != nil {
					fmt.Printf("failed to generate CSV file and skip %s\n", err)
				}
				fmt.Printf("Measurement saved in CSV file %s\n", csvPath)

				jsonPath := filepath.Join(outputLocation, fmt.Sprintf("%s_%s", current.Format(DateFormatString), "ksvc_creation_time.json"))
				jsonData, err := json.Marshal(measureFinalResult)
				if err != nil {
					fmt.Printf("failed to generate json data and skip %s\n", err)
				}
				err = utils.GenerateJSONFile(jsonData, jsonPath)
				if err != nil {
					fmt.Printf("failed to generate json file and skip %s\n", err)
				}
				fmt.Printf("Measurement saved in JSON file %s\n", jsonPath)

				htmlPath := filepath.Join(outputLocation, fmt.Sprintf("%s_%s", current.Format(DateFormatString), "ksvc_creation_time.html"))
				err = utils.GenerateHTMLFile(csvPath, htmlPath)
				if err != nil {
					fmt.Printf("failed to generate HTML file and skip %s\n", err)
				}
				fmt.Printf("Visualized measurement saved in HTML file %s\n", htmlPath)
			} else {
				fmt.Printf("-----------------------------\n")
				fmt.Printf("Basic Information:\n")
				fmt.Printf("  - Knative Versions:\n")
				fmt.Printf("    Serving: %v\n", measureFinalResult.KnativeInfo.ServingVersion)
				fmt.Printf("    Eventing: %v\n", measureFinalResult.KnativeInfo.EventingVersion)
				fmt.Printf("  - Ingress Information:\n")
				fmt.Printf("    Controller: %v\n", measureFinalResult.KnativeInfo.IngressController)
				fmt.Printf("    Version: %v\n", measureFinalResult.KnativeInfo.IngressVersion)
				fmt.Printf("Service Ready Measurement:\n")
				fmt.Printf("Total: %d | Ready: %d NotReady: %d NotFound: %d Fail: %d\n", total, measureFinalResult.Service.ReadyCount, measureFinalResult.Service.NotReadyCount, measureFinalResult.Service.NotFoundCount, measureFinalResult.Service.FailCount)
			}

			return nil
		},
	}

	serviceMeasureCommand.Flags().StringVarP(&measureArgs.svcRange, "range", "r", "", "Desired service range")
	serviceMeasureCommand.Flags().StringVarP(&measureArgs.namespace, "namespace", "", "", "Service namespace")
	serviceMeasureCommand.Flags().StringVarP(&measureArgs.svcPrefix, "svc-prefix", "", "", "Service name prefix")
	serviceMeasureCommand.Flags().BoolVarP(&measureArgs.verbose, "verbose", "v", false, "Verbose output. The details of Knative Services measuring output")
	serviceMeasureCommand.Flags().StringVarP(&measureArgs.namespaceRange, "namespace-range", "", "", "Service namespace range")
	serviceMeasureCommand.Flags().StringVarP(&measureArgs.namespacePrefix, "namespace-prefix", "", "", "Service namespace prefix")
	serviceMeasureCommand.Flags().IntVarP(&measureArgs.concurrency, "concurrency", "c", 10, "Number of workers to do measurement job")
	serviceMeasureCommand.Flags().StringVarP(&measureArgs.output, "output", "o", ".", "Measure result location")
	return serviceMeasureCommand
}

func sortSlice(rows [][]string) {
	sort.Slice(rows, func(i, j int) bool {
		a := strings.Split(rows[i][0], "-")
		indexa, _ := strconv.ParseInt(a[1], 10, 64)

		b := strings.Split(rows[j][0], "-")
		indexb, _ := strconv.ParseInt(b[1], 10, 64)
		return indexa < indexb
	})
}

// getPodCondition extracts the provided condition from the given status and returns that.
// Returns nil and -1 if the condition is not present, and the index of the located condition.
func getPodCondition(status *corev1.PodStatus, conditionType corev1.PodConditionType) (int, *corev1.PodCondition) {
	if status == nil {
		return -1, nil
	}
	for i := range status.Conditions {
		if status.Conditions[i].Type == conditionType {
			return i, &status.Conditions[i]
		}
	}
	return -1, nil
}

// getPodCondition extracts the provided condition from the given status and returns that.
// Returns nil and -1 if the condition is not present, and the index of the located condition.
func getContainerStatus(status []corev1.ContainerStatus, name string) (*corev1.ContainerStatus, bool) {
	for i := range status {
		s := &status[i]
		if s.Name == name {
			return s, true
		}
	}
	return nil, false
}

// Get Knative Serving and Eventing version
// Returns a map like {"eventing":"0.20.0", "serving":"0.20.0"}
func getKnativeVersion(p *pkg.PerfParams) map[string]string {
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
func getIngressController(p *pkg.PerfParams) map[string]string {
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
