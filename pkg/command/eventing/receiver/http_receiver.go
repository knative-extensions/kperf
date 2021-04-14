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

package receiver

import (
	"fmt"
	"log"
	"net/http"
	"runtime"
	"strconv"

	"knative.dev/kperf/pkg/command/eventing/util"
)

type ReceivedEventsStats struct {
	eventsCount          int64
	eventsLatencySeconds *[]float64 //list of latencies for received events
}

type EventsMetrics struct {
	events_total                   int64
	events_latency_seconds_buckets map[float64]int64
}

func aggegateMetrics(eventsMetrics *EventsMetrics, respChan chan ReceivedEventsStats) {
	for {
		r, ok := <-respChan
		if ok {
			// increase event count
			eventsMetrics.events_total += r.eventsCount
			// put latencies into right bucket
			// TODO
			eventsMetrics.events_latency_seconds_buckets[0.005] += 1
		}
	}
}

//TODO const BUCKETS .005, .01, .025, .05, .075, .1, .25, .5, .75, 1.0, 2.5, 5.0, 7.5, 10.0

func ReceiverRun() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	fmt.Printf("Receiver starting ...\n")
	respChan := make(chan ReceivedEventsStats, 10*1000) // make size configurable based on concurrency?

	requestHandler := func(res http.ResponseWriter, req *http.Request) {
		//data := []byte("{'Hello' :  'World!'}")
		//res.Header().Set("Content-Type", "application/json")
		res.WriteHeader(http.StatusOK)
		//res.Write(data)
		latencies := make([]float64, 1)
		//TODO calculate latency based on received timestamp
		latencies[0] = 0.01
		stats := ReceivedEventsStats{1, &latencies}
		respChan <- stats
	}

	if util.GetEnv("KAFKA_BOOTSTRAP_SERVERS", "") != "" {
		go KafkaReceive(respChan)
	}

	if util.GetEnv("REDIS_ADDRESS", "") != "" {
		go RedisReceive(respChan)
	}

	eventsMetrics := EventsMetrics{0, make(map[float64]int64, 16)}
	//eventsMetrics.events_latency_seconds_buckets[0.005] = 1
	//var eventCount
	//eventCount := int64(0)
	metricsHandler := func(res http.ResponseWriter, req *http.Request) {
		res.Header().Set("Content-Type", "text/plain")
		res.WriteHeader(http.StatusOK)
		events_total := eventsMetrics.events_total
		//TODO better str ops
		str := "events_total " + strconv.FormatInt(events_total, 10) + ".0\n"
		value := eventsMetrics.events_latency_seconds_buckets[0.005]
		str2 := str + "events_latency_seconds_bucket{le=\"0.01\"} " + strconv.FormatInt(value, 10) + ".0\n"
		data := []byte(str2)
		res.Write(data)
	}

	go aggegateMetrics(&eventsMetrics, respChan)

	// s := &http.Server{
	// 	Handler: requestHandler,
	// 	Name:    "Test Receiver",
	// }
	http.HandleFunc("/", requestHandler)
	http.HandleFunc("/metrics", metricsHandler)

	//loc := "127.0.0.1:8001"
	loc := ":8001"
	fmt.Printf("Starting receiver HTTP server on %s\n", loc)
	if err := http.ListenAndServe(loc, nil); err != nil {
		log.Fatalf("error in ListenAndServe: %s", err)
	}

}
