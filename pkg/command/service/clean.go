package service

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"

	servingv1client "knative.dev/serving/pkg/client/clientset/versioned/typed/serving/v1"

	"github.com/spf13/cobra"

	"github.com/zhanggbj/kperf/pkg"
	"github.com/zhanggbj/kperf/pkg/generator"
)

func NewServiceCleanCommand(p *pkg.PerfParams) *cobra.Command {
	ksvcCleanCommand := &cobra.Command{
		Use:   "clean",
		Short: "clean ksvc",
		Long: `clean ksvc workload

For example:
# To clean ksvc workload
kperf service clean --nsprefix testns/ --ns nsname
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
				if start >= 0 && end >= 0 && start <= end {
					for i := start; i <= end; i++ {
						nsRangeMap[fmt.Sprintf("%s%d", nsPrefix, i)] = true
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
			matchedNsNameList := [][2]string{}
			for i := 0; i < len(nsNameList); i++ {
				svcList, err := ksvcClient.Services(nsNameList[i]).List(metav1.ListOptions{})
				if err == nil {
					for j := 0; j < len(svcList.Items); j++ {
						if strings.HasPrefix(svcList.Items[j].Name, svcPrefix) {
							matchedNsNameList = append(matchedNsNameList, [2]string{nsNameList[i], svcList.Items[j].Name})
						}
					}
				}
			}
			if len(matchedNsNameList) > 0 {
				generator.NewBatchCleaner(matchedNsNameList, concurrency, cleanKsvc).Clean()
			} else {
				fmt.Printf("No service found for cleaning\n")
			}
			return nil
		},
	}

	ksvcCleanCommand.Flags().StringVarP(&nsPrefix, "nsPrefix", "p", "", "Namespace prefix. The ksvc in namespaces with the prefix will be cleaned.")
	ksvcCleanCommand.Flags().StringVarP(&nsRange, "nsRange", "", "", "")
	ksvcCleanCommand.Flags().StringVarP(&ns, "ns", "n", "", "Namespace name. The ksvc in the namespace will be cleaned.")
	ksvcCleanCommand.Flags().StringVarP(&svcPrefix, "svcPrefix", "", "testksvc", "ksvc name prefix. The ksvcs will be svcPrefix1,svcPrefix2,svcPrefix3......")
	ksvcCleanCommand.Flags().IntVarP(&concurrency, "concurrency", "c", 10, "Number of multiple ksvcs to make at a time")

	return ksvcCleanCommand
}

func cleanKsvc(ns, name string) {
	fmt.Printf("Delete ksvc %s in namespace %s\n", ns, name)
	err = ksvcClient.Services(ns).Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		fmt.Printf("Failed to delete ksvc %s in namespace %s\n", name, ns)
	}
}
