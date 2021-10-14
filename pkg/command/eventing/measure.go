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
	"bufio"
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"

	"knative.dev/kperf/pkg"
	"knative.dev/kperf/pkg/command/eventing/util"

	"github.com/spf13/cobra"

	"io/ioutil"
	"log"
	"net/http"
	"time"

	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
)

//TODO aggregate measurements

//TODO convert measurements to pretty text

//TODO write measurememnts to .csv file

func mesaureOne(t *http.Transport, metricLabels string) float64 {
	//fmt.Printf("measure\n")
	loc := util.GetEnv("METRICS_LOC1", "http://127.0.0.1:8001/metrics")
	//fmt.Printf("Getting metrics from %s\n", loc)
	req, err := http.NewRequest("GET", loc, nil)
	if err != nil {
		log.Println(err)
	}
	res, err := t.RoundTrip(req)
	if err != nil {
		log.Println(err)
		return -1
	}
	//fmt.Printf("res: %v\n", res)
	defer res.Body.Close()
	if res.StatusCode == http.StatusOK {
		bodyBytes, err := ioutil.ReadAll(res.Body)
		if err != nil {
			log.Fatal(err)
		}
		bodyString := string(bodyBytes)
		//log.Print(bodyString)
		return parseEventsTotal(bodyString, metricLabels)
	}
	return -1
}

func parseEventsTotal(metrics string, metricLabels string) float64 {
	scanner := bufio.NewScanner(strings.NewReader(metrics))
	// events_total
	key := "events_count" + metricLabels
	for scanner.Scan() {
		line := scanner.Text()
		words := strings.Fields(line)
		if len(words) > 1 {
			if words[0] == key {
				val, err := strconv.ParseFloat(words[1], 64)
				if err != nil {
					return -1
				}
				//fmt.Println(val)
				return val
			}
		}

	}
	return 0
}

//const idleSeconds = 60

func measure() {
	t := &http.Transport{}
	var prev int = -1
	var noChangeCount int
	var maxEventsPerSecond int
	idleSeconds := util.GetEnvFloat64("KPERF_EVENTING_MEASURE_IDLE_WAIT", "60.0")
	nonstop := util.GetEnv("CONTINOUS", "false") == "true" || idleSeconds <= 0
	if nonstop {
		fmt.Println("CONTINOUS mode enabled")
	}
	experimentId := util.RequiredGetEnv("EXPERIMENT_ID")
	setupId := util.RequiredGetEnv("SETUP_ID")
	workloadId := util.RequiredGetEnv("WORKLOAD_ID")
	metricLabels := fmt.Sprintf("{experimentId=\"%s\",setupId=\"%s\",workloadId=\"%s\"}", experimentId, setupId, workloadId)

	start := util.GetEnvInt("START", "500")
	concurrent := util.GetEnvInt("CONCURRENT", "1")
	durationSeconds := util.GetEnvFloat64("DURATION", "0.01")
	eventsToReceive := int(float64(start) * float64(concurrent) * durationSeconds)
	fmt.Printf("Expected to receive %d events for experimentId=%s,setupId=%s,workloadId=%s\n", eventsToReceive, experimentId, setupId, workloadId)
	logsDir := util.GetEnv("LOGS_DIR", "logs")
	path := logsDir + "/results.csv"
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
	//createHeader
	if err != nil {
		if file, err = os.Create(path); err != nil {
			log.Fatal(err)
		}
	}
	defer file.Close()
	csvWriter := csv.NewWriter(file)

	// var lastStartTime time.Time
	// var lastCount int = -1
	eventsMeasured := 0
	measureStartTime := time.Now()

	for {
		startTime := time.Now()
		val := int(mesaureOne(t, metricLabels))
		endTime := time.Now()
		timeStr := endTime.Format("20060102150405")
		if val == prev {
			noChangeCount++
		} else if prev < 0 {
			fmt.Printf("%s starting with total %d\n", timeStr, val)
		} else if val > 0 && prev > 0 {
			noChangeCount = 0
			diff := val - prev
			if diff > maxEventsPerSecond {
				maxEventsPerSecond = diff
				fmt.Printf("\n%s new maximum events per second %d", timeStr, maxEventsPerSecond)
			}
			eventsMeasured += diff
			fmt.Printf("\n%s events per second %d (total %d so far %d)\n", timeStr, diff, val, eventsMeasured)
		}
		gotAllEvent := eventsMeasured >= eventsToReceive
		if gotAllEvent || (float64(noChangeCount) > idleSeconds) {
			if gotAllEvent || !nonstop {
				if gotAllEvent {
					fmt.Printf("\nReceived %d out of expected %d\n", eventsMeasured, eventsToReceive)
				} else {
					fmt.Printf("\nNo change for %f seconds, exiting\n", idleSeconds)
				}
				measureEndTime := time.Now()
				measureDuration := measureEndTime.Sub(measureStartTime)
				measureTimeSeconds := float64(measureDuration.Nanoseconds()) / float64(time.Second)
				fmt.Printf("Measured %d events in %f seconds\n", eventsMeasured, measureTimeSeconds)
				avgEventsPerSecond := float64(eventsMeasured) / measureTimeSeconds
				fmt.Printf("Measured average events per second %f\n", avgEventsPerSecond)
				fmt.Printf("Measured maximum events per second %d\n", maxEventsPerSecond)
				maxEventsPerSecondStr := strconv.Itoa(maxEventsPerSecond)
				currentTime := time.Now()
				currentTimeYMD := currentTime.Format("20060102150405")
				row := []string{currentTimeYMD, experimentId, setupId, workloadId, maxEventsPerSecondStr}
				csvWriter.Write(row)
				csvWriter.Flush()
				os.Exit(0)
			}
			noChangeCount = 0
		}
		prev = val
		targetEndTime := startTime.Add(time.Duration(1) * time.Second)
		//duration := endTime.Sub(startTime)
		if endTime.Before(targetEndTime) {
			sleepDuration := targetEndTime.Sub(endTime)
			//fmt.Printf("%s sleeping %s\n", timeStr, sleepDuration)
			fmt.Print(".")
			time.Sleep(sleepDuration)
		}
		//time.Sleep(1 * time.Second)
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

			fmt.Printf("Eventing measure starting\n")
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
