package service

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"

	knativeapis "knative.dev/pkg/apis"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	servingv1client "knative.dev/serving/pkg/client/clientset/versioned/typed/serving/v1"

	"github.com/spf13/cobra"

	"github.com/zhanggbj/kperf/pkg"
	"github.com/zhanggbj/kperf/pkg/generator"
)

var (
	count, interval, batch, concurrency, minScale, maxScale int
	nsPrefix, nsRange, ns                                   string
	svcNamePrefixDefault                                    string = "testksvc"
	checkReady                                              bool
	timeout                                                 time.Duration
	ksvcClient                                              *servingv1client.ServingV1Client
	err                                                     error
)

func NewServiceGenerateCommand(p *pkg.PerfParams) *cobra.Command {
	ksvcGenCommand := &cobra.Command{
		Use:   "generate",
		Short: "generate ksvc",
		Long: `generate ksvc workload

For example:
# To generate ksvc workload
kperf service generate —count 500 —interval 20 —batch 20 --min-scale 0 --max-scale 5 (--nsprefix testns/ --ns nsname)
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var nsRangeMap map[string]bool = map[string]bool{}
			if nsPrefix != "" {
				r := strings.Split(nsRange, ",")
				if len(r) != 2 {
					fmt.Printf("Expected Range like 1,500, given %s\n", nsRange)
					os.Exit(1)
				}
				start, _ := strconv.Atoi(r[0])
				end, _ := strconv.Atoi(r[1])
				if start > 0 && end > 0 && start <= end {
					for i := start; i <= end; i++ {
						nsRangeMap[fmt.Sprintf("%s-%d", nsPrefix, i)] = true
					}
				}
			}

			restConfig, err := p.RestConfig()
			if err != nil {
				return err
			}
			ksvcClient, err = servingv1client.NewForConfig(restConfig)
			if err != nil {
				return err
			}
			nsNameList := []string{}
			if ns != "" {
				nss, err := p.ClientSet.CoreV1().Namespaces().Get(ns, metav1.GetOptions{})
				if err != nil {
					return err
				}
				nsNameList = append(nsNameList, nss.Name)
			} else if nsPrefix != "" {
				nsList, err := p.ClientSet.CoreV1().Namespaces().List(metav1.ListOptions{})
				if err != nil {
					return err
				}
				if len(nsList.Items) == 0 {
					return fmt.Errorf("no namespace found with prefix %s", nsPrefix)
				}
				if len(nsRangeMap) >= 0 {
					for i := 0; i < len(nsList.Items); i++ {
						if _, exists := nsRangeMap[nsList.Items[i].Name]; exists {
							nsNameList = append(nsNameList, nsList.Items[i].Name)
						}
					}
				} else {
					for i := 0; i < len(nsList.Items); i++ {
						if strings.HasPrefix(nsList.Items[i].Name, nsPrefix) {
							nsNameList = append(nsNameList, nsList.Items[i].Name)
						}
					}
				}

				if len(nsNameList) == 0 {
					return fmt.Errorf("no namespace found with prefix %s", nsPrefix)
				}
			} else {
				return fmt.Errorf("both ns and nsPrefix are empty")
			}
			if checkReady {
				generator.NewBatchGenerator(time.Duration(interval)*time.Second, count, batch, concurrency, nsNameList, createKSVC, checkServiceStatusReady).Generate()
			} else {
				generator.NewBatchGenerator(time.Duration(interval)*time.Second, count, batch, concurrency, nsNameList, createKSVC, func(ns, name string) error { return nil }).Generate()
			}

			return nil
		},
	}
	// count, interval, batch, minScale, maxScale int
	ksvcGenCommand.Flags().IntVarP(&count, "number", "n", 0, "Total number of ksvc to be created")
	ksvcGenCommand.MarkFlagRequired("count")
	ksvcGenCommand.Flags().IntVarP(&interval, "interval", "i", 0, "Interval for each batch generation")
	ksvcGenCommand.MarkFlagRequired("interval")
	ksvcGenCommand.Flags().IntVarP(&batch, "batch", "b", 0, "Number of ksvc each time to be created")
	ksvcGenCommand.MarkFlagRequired("batch")
	ksvcGenCommand.Flags().IntVarP(&concurrency, "concurrency", "c", 10, "Number of multiple ksvcs to make at a time")
	// ksvcGenCommand.MarkFlagRequired("concurrency")
	ksvcGenCommand.Flags().IntVarP(&minScale, "minScale", "", 0, "For autoscaling.knative.dev/minScale")
	ksvcGenCommand.MarkFlagRequired("minScale")
	ksvcGenCommand.Flags().IntVarP(&maxScale, "maxScale", "", 0, "For autoscaling.knative.dev/minScale")
	ksvcGenCommand.MarkFlagRequired("minScale")

	ksvcGenCommand.Flags().StringVarP(&nsPrefix, "nsPrefix", "p", "", "Namespace prefix. The ksvc will be created in the namespaces with the prefix")
	ksvcGenCommand.Flags().StringVarP(&nsRange, "nsRange", "", "", "")
	ksvcGenCommand.Flags().StringVarP(&ns, "ns", "", "", "Namespace name. The ksvc will be created in the namespace")

	ksvcGenCommand.Flags().StringVarP(&svcPrefix, "svcPrefix", "", "testksvc", "ksvc name prefix. The ksvcs will be svcPrefix1,svcPrefix2,svcPrefix3......")
	ksvcGenCommand.Flags().BoolVarP(&checkReady, "wait", "", false, "whether wait the previous ksvc to be ready")
	ksvcGenCommand.Flags().DurationVarP(&timeout, "timeout", "", 10*time.Minute, "duration to wait for previous ksvc to be ready")

	return ksvcGenCommand
}

func createKSVC(ns string, index int) (string, string) {
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
				"autoscaling.knative.dev/minScale": fmt.Sprintf("%d", minScale),
				"autoscaling.knative.dev/maxScale": fmt.Sprintf("%d", maxScale),
			},
		},
	}
	service.Spec.Template.Spec.Containers = []corev1.Container{
		{
			Image: "docker.io/qibobo/go4autoscaler:latest",
			Ports: []corev1.ContainerPort{
				{
					ContainerPort: 8080,
				},
			},
		},
	}
	fmt.Printf("Creating ksvc %s in namespace %s\n", service.GetName(), service.GetNamespace())
	_, err := ksvcClient.Services(ns).Create(&service)
	if err != nil {
		fmt.Printf("Failed to create ksvc %s in namespace %s : %s\n", service.GetName(), service.GetNamespace(), err)
	}
	return service.GetNamespace(), service.GetName()
}
func checkServiceStatusReady(ns, name string) error {
	start := time.Now()
	for time.Now().Sub(start) < timeout {
		svc, _ := ksvcClient.Services(ns).Get(name, metav1.GetOptions{})
		conditions := svc.Status.Conditions
		for i := 0; i < len(conditions); i++ {
			if conditions[i].Type == knativeapis.ConditionReady && conditions[i].IsTrue() {
				return nil
			}
		}
	}
	fmt.Printf("Error: ksvc %s in namespace %s is not ready after %s\n", name, ns, timeout)
	return fmt.Errorf("ksvc %s in namespace %s is not ready after %s ", name, ns, timeout)

}
