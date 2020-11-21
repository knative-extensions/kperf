// Copyright © 2020 The Knative Authors
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

	"knative.dev/kperf/pkg"

	"github.com/montanaflynn/stats"
	"github.com/spf13/cobra"

	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"

	corev1 "k8s.io/api/core/v1"
	podutil "k8s.io/kubernetes/pkg/api/v1/pod"
	networkingv1api "knative.dev/networking/pkg/apis/networking/v1alpha1"
	servingv1api "knative.dev/serving/pkg/apis/serving/v1"
	v1 "knative.dev/serving/pkg/apis/serving/v1"
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
					svcList := &servingv1api.ServiceList{}
					if svcList, err = servingClient.Services(svcNsName).List(context.TODO(), metav1.ListOptions{}); err != nil {
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

			rows := make([][]string, 0)
			rawRows := make([][]string, 0)

			nwclient, err := p.NewNetworkingClient()
			if err != nil {
				return fmt.Errorf("failed to create networking client%s\n", err)
			}

			svcChannel := make(chan []string)
			group := sync.WaitGroup{}
			workerMeasureResults := make([]measureResult, measureArgs.concurrency, measureArgs.concurrency)
			for i := 0; i < measureArgs.concurrency; i++ {
				workerMeasureResults[i] = measureResult{
					svcReadyTime: make([]float64, 0),
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
						}
						svc := j[0]
						svcNs := j[1]
						svcIns, err := servingClient.Services(svcNs).Get(context.TODO(), svc, metav1.GetOptions{})
						if err != nil {
							fmt.Printf("failed to get Knative Service %s\n", err)
						}
						if !svcIns.IsReady() {
							fmt.Printf("service %s/%s not ready and skip measuring\n", svc, svcNs)
							currentMeasureResult.notReadyCount++
							group.Done()
							continue
						}
						currentMeasureResult.readyCount++
						svcCreatedTime := svcIns.GetCreationTimestamp().Rfc3339Copy()
						svcConfigurationsReady := svcIns.Status.GetCondition(servingv1api.ServiceConditionConfigurationsReady).LastTransitionTime.Inner.Rfc3339Copy()
						svcRoutesReady := svcIns.Status.GetCondition(servingv1api.ServiceConditionRoutesReady).LastTransitionTime.Inner.Rfc3339Copy()

						svcConfigurationsReadyDuration = svcConfigurationsReady.Sub(svcCreatedTime.Time)
						svcRoutesReadyDuration = svcRoutesReady.Sub(svcCreatedTime.Time)
						svcReadyDuration = svcRoutesReady.Sub(svcCreatedTime.Time)

						cfgIns, err := servingClient.Configurations(svcNs).Get(context.TODO(), svc, metav1.GetOptions{})
						if err != nil {
							fmt.Printf("failed to get Configuration and skip measuring %s\n", err)
							currentMeasureResult.notReadyCount++
							group.Done()
							continue
						}
						revisionName := cfgIns.Status.LatestReadyRevisionName

						revisionIns, err := servingClient.Revisions(svcNs).Get(context.TODO(), revisionName, metav1.GetOptions{})
						if err != nil {
							fmt.Printf("failed to get Revision and skip measuring %s\n", err)
							currentMeasureResult.notReadyCount++
							group.Done()
							continue
						}

						revisionCreatedTime := revisionIns.GetCreationTimestamp().Rfc3339Copy()
						revisionReadyTime := revisionIns.Status.GetCondition(v1.RevisionConditionReady).LastTransitionTime.Inner.Rfc3339Copy()
						revisionReadyDuration := revisionReadyTime.Sub(revisionCreatedTime.Time)

						label := fmt.Sprintf("serving.knative.dev/revision=%s", revisionName)
						podList := &corev1.PodList{}
						if podList, err = p.ClientSet.CoreV1().Pods(svcNs).List(context.TODO(), metav1.ListOptions{LabelSelector: label}); err != nil {
							fmt.Errorf("list Pods of revision[%s] error :%v", revisionName, err)
							currentMeasureResult.notReadyCount++
							group.Done()
							continue
						}

						deploymentName := revisionName + "-deployment"
						deploymentIns, err := p.ClientSet.AppsV1().Deployments(svcNs).Get(context.TODO(), deploymentName, metav1.GetOptions{})
						if err != nil {
							fmt.Printf("failed to find deployment of revision[%s] error:%v", revisionName, err)
							currentMeasureResult.notReadyCount++
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
							present, PodScheduledCdt := podutil.GetPodCondition(&pod.Status, corev1.PodScheduled)
							if present == -1 {
								fmt.Printf("failed to find Pod Condition PodScheduled and skip measuring")
								currentMeasureResult.notReadyCount++
								group.Done()
								continue
							}
							podScheduledTime = PodScheduledCdt.LastTransitionTime.Rfc3339Copy()
							present, containersReadyCdt := podutil.GetPodCondition(&pod.Status, corev1.ContainersReady)
							if present == -1 {
								fmt.Printf("failed to find Pod Condition ContainersReady and skip measuring")
								currentMeasureResult.notReadyCount++
								group.Done()
								continue
							}
							containersReadyTime = containersReadyCdt.LastTransitionTime.Rfc3339Copy()
							podScheduledDuration = podScheduledTime.Sub(podCreatedTime.Time)
							containersReadyDuration = containersReadyTime.Sub(podCreatedTime.Time)

							queueProxyStatus, found := podutil.GetContainerStatus(pod.Status.ContainerStatuses, "queue-proxy")
							if !found {
								fmt.Printf("failed to get queue-proxy container status and skip, error:%v", err)
								currentMeasureResult.notReadyCount++
								group.Done()
								continue
							}
							queueProxyStartedTime = queueProxyStatus.State.Running.StartedAt.Rfc3339Copy()

							userContrainerStatus, found := podutil.GetContainerStatus(pod.Status.ContainerStatuses, "user-container")
							if !found {
								fmt.Printf("failed to get user-container container status and skip, error:%v", err)
								currentMeasureResult.notReadyCount++
								group.Done()
								continue
							}
							userContrainerStartedTime = userContrainerStatus.State.Running.StartedAt.Rfc3339Copy()

							queueProxyStartedDuration = queueProxyStartedTime.Sub(podCreatedTime.Time)
							userContrainerStartedDuration = userContrainerStartedTime.Sub(podCreatedTime.Time)
						}
						// TODO: Need to figure out a better way to measure PA time as its status keeps changing even after service creation.

						ingressIns, err := nwclient.Ingresses(svcNs).Get(context.TODO(), svc, metav1.GetOptions{})
						if err != nil {
							fmt.Printf("failed to get Ingress %s\n", err)
							currentMeasureResult.notReadyCount++
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
						rows = append(rows, []string{svc, svcNs,
							fmt.Sprintf("%d", int(svcConfigurationsReadyDuration.Seconds())),
							fmt.Sprintf("%d", int(revisionReadyDuration.Seconds())),
							fmt.Sprintf("%d", int(deploymentCreatedDuration.Seconds())),
							fmt.Sprintf("%d", int(podScheduledDuration.Seconds())),
							fmt.Sprintf("%d", int(containersReadyDuration.Seconds())),
							fmt.Sprintf("%d", int(queueProxyStartedDuration.Seconds())),
							fmt.Sprintf("%d", int(userContrainerStartedDuration.Seconds())),
							fmt.Sprintf("%d", int(svcRoutesReadyDuration.Seconds())),
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
							ingressCreatedTime.String(),
							ingressNetworkConfiguredTime.String(),
							ingressLoadBalancerReadyTime.String()})

						if cmd.Flags().Changed("verbose") {
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

						currentMeasureResult.svcConfigurationsReadySum += svcConfigurationsReadyDuration.Seconds()
						currentMeasureResult.revisionReadySum += revisionReadyDuration.Seconds()
						currentMeasureResult.deploymentCreatedSum += deploymentCreatedDuration.Seconds()
						currentMeasureResult.podScheduledSum += podScheduledDuration.Seconds()
						currentMeasureResult.containersReadySum += containersReadyDuration.Seconds()
						currentMeasureResult.queueProxyStartedSum += queueProxyStartedDuration.Seconds()
						currentMeasureResult.userContrainerStartedSum += userContrainerStartedDuration.Seconds()
						currentMeasureResult.svcRoutesReadySum += svcRoutesReadyDuration.Seconds()
						currentMeasureResult.ingressReadySum += ingressReadyDuration.Seconds()
						currentMeasureResult.ingressNetworkConfiguredSum += ingressNetworkConfiguredDuration.Seconds()
						currentMeasureResult.ingressLoadBalancerReadySum += ingressLoadBalancerReadyDuration.Seconds()
						currentMeasureResult.svcReadySum += svcReadyDuration.Seconds()
						currentMeasureResult.svcReadyTime = append(currentMeasureResult.svcReadyTime, svcReadyDuration.Seconds())
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
				measureFinalResult.svcConfigurationsReadySum += workerMeasureResults[i].svcConfigurationsReadySum
				measureFinalResult.revisionReadySum += workerMeasureResults[i].revisionReadySum
				measureFinalResult.deploymentCreatedSum += workerMeasureResults[i].deploymentCreatedSum
				measureFinalResult.podScheduledSum += workerMeasureResults[i].podScheduledSum
				measureFinalResult.containersReadySum += workerMeasureResults[i].containersReadySum
				measureFinalResult.queueProxyStartedSum += workerMeasureResults[i].queueProxyStartedSum
				measureFinalResult.userContrainerStartedSum += workerMeasureResults[i].userContrainerStartedSum
				measureFinalResult.svcRoutesReadySum += workerMeasureResults[i].svcRoutesReadySum
				measureFinalResult.ingressReadySum += workerMeasureResults[i].ingressReadySum
				measureFinalResult.ingressNetworkConfiguredSum += workerMeasureResults[i].ingressNetworkConfiguredSum
				measureFinalResult.ingressLoadBalancerReadySum += workerMeasureResults[i].ingressLoadBalancerReadySum
				measureFinalResult.svcReadyTime = append(measureFinalResult.svcReadyTime, workerMeasureResults[i].svcReadyTime...)
				measureFinalResult.svcReadySum += workerMeasureResults[i].svcReadySum
				measureFinalResult.readyCount += workerMeasureResults[i].readyCount
				measureFinalResult.notReadyCount += workerMeasureResults[i].notReadyCount
				measureFinalResult.notFoundCount += workerMeasureResults[i].notFoundCount
			}

			sortSlice(rows)
			sortSlice(rawRows)

			rows = append([][]string{{"svc_name", "svc_namespace", "configuration_ready", "revision_ready",
				"deployment_created", "pod_scheduled", "containers_ready", "queue-proxy_started", "user-container_started",
				"route_ready", "ingress_ready", "ingress_config_ready", "ingress_lb_ready", "overall_ready"}}, rows...)

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
				"ingress_created",
				"ingress_config_ready",
				"ingress_lb_ready"}}, rawRows...)
			total := measureFinalResult.readyCount + measureFinalResult.notReadyCount + measureFinalResult.notFoundCount
			if measureFinalResult.readyCount > 0 {
				fmt.Printf("-------- Measurement --------\n")
				fmt.Printf("Total: %d | Ready: %d Fail: %d NotFound: %d \n", total, measureFinalResult.readyCount, measureFinalResult.notReadyCount, measureFinalResult.notFoundCount)
				fmt.Printf("Service Configuration Duration:\n")
				fmt.Printf("Total: %fs\n", float64(measureFinalResult.svcConfigurationsReadySum))
				fmt.Printf("Average: %fs\n", float64(measureFinalResult.svcConfigurationsReadySum)/float64(measureFinalResult.readyCount))

				fmt.Printf("- Service Revision Duration:\n")
				fmt.Printf("  Total: %fs\n", float64(measureFinalResult.revisionReadySum))
				fmt.Printf("  Average: %fs\n", float64(measureFinalResult.revisionReadySum)/float64(measureFinalResult.readyCount))

				fmt.Printf("  - Service Deployment Created Duration:\n")
				fmt.Printf("    Total: %fs\n", float64(measureFinalResult.deploymentCreatedSum))
				fmt.Printf("    Average: %fs\n", float64(measureFinalResult.deploymentCreatedSum)/float64(measureFinalResult.readyCount))

				fmt.Printf("    - Service Pod Scheduled Duration:\n")
				fmt.Printf("      Total: %fs\n", float64(measureFinalResult.podScheduledSum))
				fmt.Printf("      Average: %fs\n", float64(measureFinalResult.podScheduledSum)/float64(measureFinalResult.readyCount))

				fmt.Printf("    - Service Pod Containers Ready Duration:\n")
				fmt.Printf("      Total: %fs\n", float64(measureFinalResult.containersReadySum))
				fmt.Printf("      Average: %fs\n", float64(measureFinalResult.containersReadySum)/float64(measureFinalResult.readyCount))

				fmt.Printf("      - Service Pod queue-proxy Started Duration:\n")
				fmt.Printf("        Total: %fs\n", float64(measureFinalResult.queueProxyStartedSum))
				fmt.Printf("        Average: %fs\n", float64(measureFinalResult.queueProxyStartedSum)/float64(measureFinalResult.readyCount))

				fmt.Printf("      - Service Pod user-container Started Duration:\n")
				fmt.Printf("        Total: %fs\n", float64(measureFinalResult.userContrainerStartedSum))
				fmt.Printf("        Average: %fs\n", float64(measureFinalResult.userContrainerStartedSum)/float64(measureFinalResult.readyCount))

				fmt.Printf("\nService Route Ready Duration:\n")
				fmt.Printf("Total: %fs\n", float64(measureFinalResult.svcRoutesReadySum))
				fmt.Printf("Average: %fs\n", float64(measureFinalResult.svcRoutesReadySum)/float64(measureFinalResult.readyCount))

				fmt.Printf("- Service Ingress Ready Duration:\n")
				fmt.Printf("  Total: %fs\n", float64(measureFinalResult.ingressReadySum))
				fmt.Printf("  Average: %fs\n", float64(measureFinalResult.ingressReadySum)/float64(measureFinalResult.readyCount))

				fmt.Printf("  - Service Ingress Network Configured Duration:\n")
				fmt.Printf("    Total: %fs\n", float64(measureFinalResult.ingressNetworkConfiguredSum))
				fmt.Printf("    Average: %fs\n", float64(measureFinalResult.ingressNetworkConfiguredSum)/float64(measureFinalResult.readyCount))

				fmt.Printf("  - Service Ingress LoadBalancer Ready Duration:\n")
				fmt.Printf("    Total: %fs\n", float64(measureFinalResult.ingressLoadBalancerReadySum))
				fmt.Printf("    Average: %fs\n", float64(measureFinalResult.ingressLoadBalancerReadySum)/float64(measureFinalResult.readyCount))

				fmt.Printf("\n-----------------------------\n")
				fmt.Printf("Overall Service Ready Measurement:\n")
				fmt.Printf("Total: %d | Ready: %d Fail: %d NotFound: %d \n", total, measureFinalResult.readyCount, measureFinalResult.notReadyCount, measureFinalResult.notFoundCount)
				fmt.Printf("Total: %fs\n", measureFinalResult.svcReadySum)
				fmt.Printf("Average: %fs\n", float64(measureFinalResult.svcReadySum)/float64(measureFinalResult.readyCount))

				median, err := stats.Median(measureFinalResult.svcReadyTime)
				fmt.Printf("Median: %fs\n", median)

				min, err := stats.Min(measureFinalResult.svcReadyTime)
				fmt.Printf("Min: %fs\n", min)

				max, err := stats.Max(measureFinalResult.svcReadyTime)
				fmt.Printf("Max: %fs\n", max)

				p50, err := stats.Percentile(measureFinalResult.svcReadyTime, 50)
				fmt.Printf("Percentile50: %fs\n", p50)

				p90, err := stats.Percentile(measureFinalResult.svcReadyTime, 90)
				fmt.Printf("Percentile90: %fs\n", p90)

				p95, err := stats.Percentile(measureFinalResult.svcReadyTime, 95)
				fmt.Printf("Percentile95: %fs\n", p95)

				p98, err := stats.Percentile(measureFinalResult.svcReadyTime, 98)
				fmt.Printf("Percentile98: %fs\n", p98)

				p99, err := stats.Percentile(measureFinalResult.svcReadyTime, 99)
				fmt.Printf("Percentile99: %fs\n", p99)

				current := time.Now()
				rawPath := fmt.Sprintf("/tmp/%s_%s", current.Format("20060102150405"), "raw_ksvc_creation_time.csv")
				err = utils.GenerateCSVFile(rawPath, rawRows)
				if err != nil {
					fmt.Printf("failed to generate raw timestamp file and skip %s\n", err)
				}
				fmt.Printf("Raw Timestamp saved in CSV file %s\n", rawPath)

				csvPath := fmt.Sprintf("/tmp/%s_%s", current.Format("20060102150405"), "ksvc_creation_time.csv")
				err = utils.GenerateCSVFile(csvPath, rows)
				if err != nil {
					fmt.Printf("failed to generate CSV file and skip %s\n", err)
				}

				fmt.Printf("Measurement saved in CSV file %s\n", csvPath)
				htmlPath := fmt.Sprintf("/tmp/%s_%s", current.Format("20060102150405"), "ksvc_creation_time.html")
				err = utils.GenerateHTMLFile(csvPath, htmlPath)
				if err != nil {
					fmt.Printf("failed to generate HTML file and skip %s\n", err)
				}
				fmt.Printf("Visualized measurement saved in HTML file %s\n", htmlPath)
			} else {
				fmt.Printf("-----------------------------\n")
				fmt.Printf("Service Ready Measurement:\n")
				fmt.Printf("Total: %d | Ready: %d Fail: %d NotFound: %d \n", total, measureFinalResult.readyCount, measureFinalResult.notReadyCount, measureFinalResult.notFoundCount)
			}

			return nil
		},
	}

	serviceMeasureCommand.Flags().StringVarP(&measureArgs.svcRange, "range", "r", "", "Desired service range")
	serviceMeasureCommand.Flags().StringVarP(&measureArgs.namespace, "namespace", "", "", "Service namespace")
	serviceMeasureCommand.Flags().StringVarP(&measureArgs.svcPrefix, "svc-prefix", "", "", "Service name prefix")
	serviceMeasureCommand.Flags().BoolVarP(&measureArgs.verbose, "verbose", "v", false, "Service verbose result")
	serviceMeasureCommand.Flags().StringVarP(&measureArgs.namespaceRange, "namespace-range", "", "", "Service namespace range")
	serviceMeasureCommand.Flags().StringVarP(&measureArgs.namespacePrefix, "namespace-prefix", "", "", "Service namespace prefix")
	serviceMeasureCommand.Flags().IntVarP(&measureArgs.concurrency, "concurrency", "c", 10, "Number of workers to do measurement job")
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
