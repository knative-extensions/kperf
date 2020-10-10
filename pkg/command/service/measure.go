package service

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/zhanggbj/kperf/pkg/command/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/zhanggbj/kperf/pkg"

	"github.com/montanaflynn/stats"
	"github.com/spf13/cobra"

	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"

	corev1 "k8s.io/api/core/v1"
	podutil "k8s.io/kubernetes/pkg/api/v1/pod"
	networkingv1api "knative.dev/networking/pkg/apis/networking/v1alpha1"
	networkingv1alpha1 "knative.dev/networking/pkg/client/clientset/versioned/typed/networking/v1alpha1"
	servingv1api "knative.dev/serving/pkg/apis/serving/v1"
	v1 "knative.dev/serving/pkg/apis/serving/v1"
	servingv1client "knative.dev/serving/pkg/client/clientset/versioned/typed/serving/v1"
)

var (
	svcPrefix, svcRange, svcName, svcNamespace, svcNsRange, svcNsPrefix string
	svcConfigurationsReadySum, svcRoutesReadyReadySum, svcReadySum, minDomainReadySum, maxDomainReadySum,
	revisionReadySum, podAutoscalerReadySum, ingressReadyReadySum, ingressNetworkConfiguredSum,
	ingressLoadBalancerReadySum, podScheduledSum, containersReadySum, queueProxyStartedSum,
	userContrainerStartedSum, deploymentCreatedSum float64
	verbose                                              bool
	readyCount, notReadyCount, notFoundCount, measureJob int
	lock                                                 sync.Mutex
)

func NewServiceMeasureCommand(p *pkg.PerfParams) *cobra.Command {
	serviceMeasureCommand := &cobra.Command{
		Use:   "measure",
		Short: "Measure Knative service",
		Long: `Measure Knative service creation time

For example:
# To measure a Codeengine service creation time running currently with 20 jobs
kperf service measure --perfix svc --range 1,200 --namespace ns --job 20
`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Flags().NFlag() == 0 {
				return fmt.Errorf("'service measure' requires flag(s)")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			svcNamespacedName := make([][]string, 0)
			if cmd.Flags().Changed("namespace") {
				r := strings.Split(svcRange, ",")
				if len(r) != 2 {
					return fmt.Errorf("expected range like 1,500, given %s\n", svcRange)
				}
				start, _ := strconv.Atoi(r[0])
				end, _ := strconv.Atoi(r[1])
				for i := start; i <= end; i++ {
					sName := fmt.Sprintf("%s-%s", svcPrefix, strconv.Itoa(i))
					svcNamespacedName = append(svcNamespacedName, []string{sName, svcNamespace})
				}
			}

			cfg, _ := p.RestConfig()
			servingClient, err := servingv1client.NewForConfig(cfg)
			if err != nil {
				return fmt.Errorf("failed to create serving client%s\n", err)
			}

			if cmd.Flags().Changed("nsrange") && cmd.Flags().Changed("nsprefix") {
				r := strings.Split(svcNsRange, ",")
				if len(r) != 2 {
					return fmt.Errorf("expected nsrange like 1,500, given %s\n", svcNsRange)
				}
				start, _ := strconv.Atoi(r[0])
				end, _ := strconv.Atoi(r[1])
				for i := start; i <= end; i++ {
					svcNsName := fmt.Sprintf("%s-%s", svcNsPrefix, strconv.Itoa(i))
					svcList := &servingv1api.ServiceList{}
					if svcList, err = servingClient.Services(svcNsName).List(metav1.ListOptions{}); err != nil {
						return fmt.Errorf("failed to list service under namespace %s error:%v", svcNsName, err)
					}

					if len(svcList.Items) > 0 {
						for _, svc := range svcList.Items {
							svcNamespacedName = append(svcNamespacedName, []string{svc.Name, svcNsName})
						}
					} else {
						fmt.Printf("no service found under namespace %s and skip\n", svcNsName)
					}
				}
			}

			rows := make([][]string, 0)
			rawRows := make([][]string, 0)

			svcReadyTime := make([]float64, 0)

			client, err := kubernetes.NewForConfig(cfg)
			if err != nil {
				return fmt.Errorf("failed to create k8s client%s\n", err)
			}

			nwclient, err := networkingv1alpha1.NewForConfig(cfg)
			if err != nil {
				return fmt.Errorf("failed to create serving client%s\n", err)
			}

			svcChannel := make(chan []string)
			group := sync.WaitGroup{}

			for i := 0; i < measureJob; i++ {
				go func() {
					var (
						svcConfigurationsReadyDuration, svcReadyDuration, svcRoutesReadyDuration, podScheduledDuration,
						containersReadyDuration, queueProxyStartedDuration, userContrainerStartedDuration time.Duration
					)
					for j := range svcChannel {
						if len(j) != 2 {
							fmt.Errorf("lack of service name or service namespace and skip")
						}
						svc := j[0]
						svcNs := j[1]
						svcIns, err := servingClient.Services(svcNs).Get(svc, metav1.GetOptions{})
						if err != nil {
							fmt.Errorf("failed to get Knative Service %s\n", err)
						}
						if !svcIns.Status.IsReady() {
							fmt.Printf("service %s/%s not ready and skip measuring\n", svc, svcNs)
							notReadyCount = notReadyCount + 1
							group.Done()
							continue
						}
						readyCount = readyCount + 1
						svcCreatedTime := svcIns.GetCreationTimestamp().Rfc3339Copy()
						svcConfigurationsReady := svcIns.Status.GetCondition(servingv1api.ServiceConditionConfigurationsReady).LastTransitionTime.Inner.Rfc3339Copy()
						svcRoutesReady := svcIns.Status.GetCondition(servingv1api.ServiceConditionRoutesReady).LastTransitionTime.Inner.Rfc3339Copy()

						svcConfigurationsReadyDuration = svcConfigurationsReady.Sub(svcCreatedTime.Time)
						svcRoutesReadyDuration = svcRoutesReady.Sub(svcCreatedTime.Time)
						svcReadyDuration = svcRoutesReady.Sub(svcCreatedTime.Time)

						cfgIns, err := servingClient.Configurations(svcNs).Get(svc, metav1.GetOptions{})
						if err != nil {
							fmt.Errorf("failed to get Configuration and skip measuring %s\n", err)
							notReadyCount = notReadyCount + 1
							group.Done()
							continue
						}
						revisionName := cfgIns.Status.LatestReadyRevisionName

						revisionIns, err := servingClient.Revisions(svcNs).Get(revisionName, metav1.GetOptions{})
						if err != nil {
							fmt.Errorf("failed to get Revision and skip measuring %s\n", err)
							notReadyCount = notReadyCount + 1
							group.Done()
							continue
						}

						revisionCreatedTime := revisionIns.GetCreationTimestamp().Rfc3339Copy()
						revisionReadyTime := revisionIns.Status.GetCondition(v1.RevisionConditionReady).LastTransitionTime.Inner.Rfc3339Copy()
						revisionReadyDuration := revisionReadyTime.Sub(revisionCreatedTime.Time)

						label := fmt.Sprintf("serving.knative.dev/revision=%s", revisionName)
						podList := &corev1.PodList{}
						if podList, err = client.CoreV1().Pods(svcNs).List(metav1.ListOptions{LabelSelector: label}); err != nil {
							fmt.Errorf("list Pods of revision[%s] error :%v", revisionName, err)
							notReadyCount = notReadyCount + 1
							group.Done()
							continue
						}

						deploymentName := revisionName + "-deployment"
						deploymentIns, err := client.AppsV1().Deployments(svcNs).Get(deploymentName, metav1.GetOptions{})
						if err != nil {
							fmt.Errorf("failed to find deployment of revision[%s] error:%v", revisionName, err)
							notReadyCount = notReadyCount + 1
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
								fmt.Errorf("failed to find Pod Condition PodScheduled and skip measuring")
								notReadyCount = notReadyCount + 1
								group.Done()
								continue
							}
							podScheduledTime = PodScheduledCdt.LastTransitionTime.Rfc3339Copy()
							present, containersReadyCdt := podutil.GetPodCondition(&pod.Status, corev1.ContainersReady)
							if present == -1 {
								fmt.Errorf("failed to find Pod Condition ContainersReady and skip measuring")
								notReadyCount = notReadyCount + 1
								group.Done()
								continue
							}
							containersReadyTime = containersReadyCdt.LastTransitionTime.Rfc3339Copy()
							podScheduledDuration = podScheduledTime.Sub(podCreatedTime.Time)
							containersReadyDuration = containersReadyTime.Sub(podCreatedTime.Time)

							queueProxyStatus, found := podutil.GetContainerStatus(pod.Status.ContainerStatuses, "queue-proxy")
							if !found {
								fmt.Errorf("failed to get queue-proxy container status and skip, error:%v", err)
								notReadyCount = notReadyCount + 1
								group.Done()
								continue
							}
							queueProxyStartedTime = queueProxyStatus.State.Running.StartedAt.Rfc3339Copy()

							userContrainerStatus, found := podutil.GetContainerStatus(pod.Status.ContainerStatuses, "user-container")
							if !found {
								fmt.Errorf("failed to get user-container container status and skip, error:%v", err)
								notReadyCount = notReadyCount + 1
								group.Done()
								continue
							}
							userContrainerStartedTime = userContrainerStatus.State.Running.StartedAt.Rfc3339Copy()

							queueProxyStartedDuration = queueProxyStartedTime.Sub(podCreatedTime.Time)
							userContrainerStartedDuration = userContrainerStartedTime.Sub(podCreatedTime.Time)
						}
						// TODO: Need to figure out a better way to measure PA time as its status keeps changing even after service creation.

						ingressIns, err := nwclient.Ingresses(svcNs).Get(svc, metav1.GetOptions{})
						if err != nil {
							fmt.Errorf("failed to get Ingress %s\n", err)
							notReadyCount = notReadyCount + 1
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

						svcConfigurationsReadySum = svcConfigurationsReadySum + svcConfigurationsReadyDuration.Seconds()
						revisionReadySum = revisionReadySum + revisionReadyDuration.Seconds()
						deploymentCreatedSum = deploymentCreatedSum + deploymentCreatedDuration.Seconds()
						podScheduledSum = podScheduledSum + podScheduledDuration.Seconds()
						containersReadySum = containersReadySum + containersReadyDuration.Seconds()
						queueProxyStartedSum = queueProxyStartedSum + queueProxyStartedDuration.Seconds()
						userContrainerStartedSum = userContrainerStartedSum + userContrainerStartedDuration.Seconds()
						svcRoutesReadyReadySum = svcRoutesReadyReadySum + svcRoutesReadyDuration.Seconds()
						ingressReadyReadySum = ingressReadyReadySum + ingressReadyDuration.Seconds()
						ingressNetworkConfiguredSum = ingressNetworkConfiguredSum + ingressNetworkConfiguredDuration.Seconds()
						ingressLoadBalancerReadySum = ingressLoadBalancerReadySum + ingressLoadBalancerReadyDuration.Seconds()
						svcReadySum = svcReadySum + svcReadyDuration.Seconds()
						svcReadyTime = append(svcReadyTime, svcReadyDuration.Seconds())
						lock.Unlock()
						group.Done()
					}
				}()
			}

			for _, item := range svcNamespacedName {
				svcChannel <- item
				group.Add(1)
			}

			group.Wait()
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
			total := readyCount + notReadyCount + notFoundCount
			if readyCount > 0 {
				fmt.Printf("-------- Measurement --------\n")
				fmt.Printf("Total: %d | Ready: %d Fail: %d NotFound: %d \n", total, readyCount, notReadyCount, notFoundCount)
				fmt.Printf("Service Configuration Duration:\n")
				fmt.Printf("Total: %fs\n", float64(svcConfigurationsReadySum))
				fmt.Printf("Average: %fs\n", float64(svcConfigurationsReadySum)/float64(readyCount))

				fmt.Printf("- Service Revision Duration:\n")
				fmt.Printf("  Total: %fs\n", float64(revisionReadySum))
				fmt.Printf("  Average: %fs\n", float64(revisionReadySum)/float64(readyCount))

				fmt.Printf("  - Service Deployment Created Duration:\n")
				fmt.Printf("    Total: %fs\n", float64(revisionReadySum))
				fmt.Printf("    Average: %fs\n", float64(revisionReadySum)/float64(readyCount))

				fmt.Printf("    - Service Pod Scheduled Duration:\n")
				fmt.Printf("      Total: %fs\n", float64(podScheduledSum))
				fmt.Printf("      Average: %fs\n", float64(podScheduledSum)/float64(readyCount))

				fmt.Printf("    - Service Pod Containers Ready Duration:\n")
				fmt.Printf("      Total: %fs\n", float64(containersReadySum))
				fmt.Printf("      Average: %fs\n", float64(containersReadySum)/float64(readyCount))

				fmt.Printf("      - Service Pod queue-proxy Started Duration:\n")
				fmt.Printf("        Total: %fs\n", float64(queueProxyStartedSum))
				fmt.Printf("        Average: %fs\n", float64(queueProxyStartedSum)/float64(readyCount))

				fmt.Printf("      - Service Pod user-container Started Duration:\n")
				fmt.Printf("        Total: %fs\n", float64(userContrainerStartedSum))
				fmt.Printf("        Average: %fs\n", float64(userContrainerStartedSum)/float64(readyCount))

				fmt.Printf("\nService Route Ready Duration:\n")
				fmt.Printf("Total: %fs\n", float64(svcRoutesReadyReadySum))
				fmt.Printf("Average: %fs\n", float64(svcRoutesReadyReadySum)/float64(readyCount))

				fmt.Printf("- Service Ingress Ready Duration:\n")
				fmt.Printf("  Total: %fs\n", float64(ingressReadyReadySum))
				fmt.Printf("  Average: %fs\n", float64(ingressReadyReadySum)/float64(readyCount))

				fmt.Printf("  - Service Ingress Network Configured Duration:\n")
				fmt.Printf("    Total: %fs\n", float64(ingressNetworkConfiguredSum))
				fmt.Printf("    Average: %fs\n", float64(ingressNetworkConfiguredSum)/float64(readyCount))

				fmt.Printf("  - Service Ingress LoadBalancer Ready Duration:\n")
				fmt.Printf("    Total: %fs\n", float64(ingressLoadBalancerReadySum))
				fmt.Printf("    Average: %fs\n", float64(ingressLoadBalancerReadySum)/float64(readyCount))

				fmt.Printf("\n-----------------------------\n")
				fmt.Printf("Overall Service Ready Measurement:\n")
				fmt.Printf("Total: %d | Ready: %d Fail: %d NotFound: %d \n", total, readyCount, notReadyCount, notFoundCount)
				fmt.Printf("Total: %fs\n", svcReadySum)
				fmt.Printf("Average: %fs\n", float64(svcReadySum)/float64(readyCount))

				median, err := stats.Median(svcReadyTime)
				fmt.Printf("Median: %fs\n", median)

				min, err := stats.Min(svcReadyTime)
				fmt.Printf("Min: %fs\n", min)

				max, err := stats.Max(svcReadyTime)
				fmt.Printf("Max: %fs\n", max)

				p50, err := stats.Percentile(svcReadyTime, 50)
				fmt.Printf("Percentile50: %fs\n", p50)

				p90, err := stats.Percentile(svcReadyTime, 90)
				fmt.Printf("Percentile90: %fs\n", p90)

				p95, err := stats.Percentile(svcReadyTime, 95)
				fmt.Printf("Percentile95: %fs\n", p95)

				p98, err := stats.Percentile(svcReadyTime, 98)
				fmt.Printf("Percentile98: %fs\n", p98)

				p99, err := stats.Percentile(svcReadyTime, 99)
				fmt.Printf("Percentile99: %fs\n", p99)

				current := time.Now()
				rawPath := fmt.Sprintf("/tmp/%s_%s", current.Format("20060102150405"), "raw_ksvc_creation_time.csv")
				err = utils.GenerateCSVFile(rawPath, rawRows)
				if err != nil {
					fmt.Errorf("failed to generate raw timestamp file and skip %s\n", err)
				}
				fmt.Printf("Raw Timestamp saved in CSV file %s\n", rawPath)

				csvPath := fmt.Sprintf("/tmp/%s_%s", current.Format("20060102150405"), "ksvc_creation_time.csv")
				err = utils.GenerateCSVFile(csvPath, rows)
				if err != nil {
					fmt.Errorf("failed to generate CSV file and skip %s\n", err)
				}

				fmt.Printf("Measurement saved in CSV file %s\n", csvPath)
				htmlPath := fmt.Sprintf("/tmp/%s_%s", current.Format("20060102150405"), "ksvc_creation_time.html")
				err = utils.GenerateHTMLFile(csvPath, htmlPath)
				if err != nil {
					fmt.Errorf("failed to generate HTML file and skip %s\n", err)
				}
				fmt.Printf("Visualized measurement saved in HTML file %s\n", htmlPath)
			} else {
				fmt.Printf("-----------------------------\n")
				fmt.Printf("Service Ready Measurement:\n")
				fmt.Printf("Total: %d | Ready: %d Fail: %d NotFound: %d \n", total, readyCount, notReadyCount, notFoundCount)
			}

			return nil
		},
	}

	serviceMeasureCommand.Flags().StringVarP(&svcRange, "range", "r", "", "Desired service range")
	serviceMeasureCommand.Flags().StringVarP(&svcNamespace, "namespace", "n", "", "Service namespace")
	serviceMeasureCommand.Flags().StringVarP(&svcPrefix, "prefix", "p", "", "Service name prefix")
	serviceMeasureCommand.Flags().BoolVarP(&verbose, "verbose", "v", false, "Service verbose result")
	serviceMeasureCommand.Flags().StringVarP(&svcNsRange, "nsrange", "", "", "Service namespace range")
	serviceMeasureCommand.Flags().StringVarP(&svcNsPrefix, "nsprefix", "", "", "Service namespace prefix")
	serviceMeasureCommand.Flags().IntVarP(&measureJob, "job", "j", 10, "Service measurement job")
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
