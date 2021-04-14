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

package eventing

import (
	"fmt"

	"knative.dev/kperf/pkg"

	"github.com/spf13/cobra"

	"io/ioutil"
	"log"
	"net/http"
	"time"

	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
)

//TODO aggregate measurements

//TODO convert measurements ot pretty text

//TODO write measurememnts ot .csv file

func mesaureOne(t *http.Transport) {
	//fmt.Printf("measure\n")
	req, err := http.NewRequest("GET", "http://127.0.0.1:8001/metrics", nil)
	if err != nil {
		log.Println(err)
	}
	res, err := t.RoundTrip(req)
	if err != nil {
		log.Println(err)
	}
	//fmt.Printf("res: %v\n", res)
	defer res.Body.Close()
	if res.StatusCode == http.StatusOK {
		bodyBytes, err := ioutil.ReadAll(res.Body)
		if err != nil {
			log.Fatal(err)
		}
		bodyString := string(bodyBytes)
		log.Print(bodyString)
	}

}

func measure() {
	t := &http.Transport{}
	for {
		mesaureOne(t)
		time.Sleep(1 * time.Second)
	}
}

const (
	DateFormatString = "20060102150405"
)

func NewEventingMeasureCommand(p *pkg.PerfParams) *cobra.Command {
	measureArgs := measureArgs{}
	//measureFinalResult := measureResult{}
	eventingMeasureCommand := &cobra.Command{
		Use:   "measure",
		Short: "Measure Knative eventing",
		Long: `Measure Knative eventing workload

For example:
# To measure a Knative Eventing workloadrunning currently with 20 concurent jobs
kperf eventing measure --svc-perfix svc --range 1,200 --namespace ns --concurrency 20
`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {

			fmt.Printf("Eventing measure starring\n")
			measure()
			return nil
		},
	}

	eventingMeasureCommand.Flags().StringVarP(&measureArgs.svcRange, "range", "r", "", "Desired service range")
	eventingMeasureCommand.Flags().StringVarP(&measureArgs.namespace, "namespace", "", "", "Service namespace")
	eventingMeasureCommand.Flags().StringVarP(&measureArgs.svcPrefix, "svc-prefix", "", "", "Service name prefix")
	eventingMeasureCommand.Flags().BoolVarP(&measureArgs.verbose, "verbose", "v", false, "Service verbose result")
	eventingMeasureCommand.Flags().StringVarP(&measureArgs.namespaceRange, "namespace-range", "", "", "Service namespace range")
	eventingMeasureCommand.Flags().StringVarP(&measureArgs.namespacePrefix, "namespace-prefix", "", "", "Service namespace prefix")
	eventingMeasureCommand.Flags().IntVarP(&measureArgs.concurrency, "concurrency", "c", 10, "Number of workers to do measurement job")
	eventingMeasureCommand.Flags().StringVarP(&measureArgs.output, "output", "o", ".", "Measure result location")
	return eventingMeasureCommand
}
