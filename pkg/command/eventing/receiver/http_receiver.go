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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"knative.dev/kperf/pkg/command/eventing/util"
)

type ReceivedEventsStats struct {
	//labels  *map[string]string
	//metricLabels string
	experimentId string
	setupId      string
	workloadId   string
	source       string
	eventId      string
	// sequence       int
	// sequencemax    int
	latencySeconds float64
	//eventsCount          int64
	//eventsLatencySeconds *[]float64 //list of latencies for received events
}

const events_metric = "events"

// map "" ->
// map labels ->

type EventsMetrics struct {
	lock *sync.RWMutex
	//events_total                   int64
	//events_total map[string]int64
	events_total map[string]*ExperimentStats
	//events_latency_seconds_buckets map[float64]int64
	// separate total and latencies for each event source
	// keep next sequence and list of received ids (for duplicates)
}

func updateMetrics(eventsMetrics *EventsMetrics, s *EventsMetrics) {
	eventsMetrics.lock.Lock()
	defer eventsMetrics.lock.Unlock()
	//eventsMetrics.events_total[""] += int64(s.events_total[""])
	//TODO add for all keys in s
	for key, val := range s.events_total {
		//eventsMetrics.events_total[key] += val
		if eventsMetrics.events_total[key] == nil {
			eventsMetrics.events_total[key] = NewExperimentStats(events_metric, key, eventsMetrics.events_total[""].GetBucketList())
		}
		//fmt.Printf("Merge val=%v\n", val)
		eventsMetrics.events_total[key].Merge(val)
	}
	//s.events_total[""] = 0
	s.events_total = make(map[string]*ExperimentStats)
	s.events_total[""] = NewExperimentStats(events_metric, "", eventsMetrics.events_total[""].GetBucketList())
}

func aggegateMetrics(eventsMetrics *EventsMetrics, respChan chan ReceivedEventsStats, ticker *time.Ticker) {
	//var localCount int64
	var s *EventsMetrics = &EventsMetrics{}
	s.events_total = make(map[string]*ExperimentStats)
	s.events_total[""] = NewExperimentStats(events_metric, "", eventsMetrics.events_total[""].GetBucketList())
	for {
		select {
		case r, ok := <-respChan:
			if ok {
				s.events_total[""].Add(r.latencySeconds)
				metricLabels := fmt.Sprintf("experimentId=\"%s\",setupId=\"%s\",workloadId=\"%s\"", r.experimentId, r.setupId, r.workloadId)
				if s.events_total[metricLabels] == nil {
					s.events_total[metricLabels] = NewExperimentStats(events_metric, metricLabels, eventsMetrics.events_total[""].GetBucketList())
				}
				s.events_total[metricLabels].Add(r.latencySeconds)
			}
		case <-ticker.C: //tm := <-ticker.C:
			//fmt.Println("Tick@", tm)
			// aggregate computed counts use mutex to transfer
			updateMetrics(eventsMetrics, s)
		}

		// _, ok := <-respChan
		// if ok {
		// 	//eventsMetrics.mu.Lock()
		// 	// increase event count
		// 	//val := eventsMetrics.events_total
		// 	//fmt.Printf("aggregating %d\n", val)
		// 	eventsMetrics.events_total++ //= r.eventsCount
		// 	// put latencies into right bucket
		// 	// TODO
		// 	//eventsMetrics.events_latency_seconds_buckets[0.005] += 1
		// 	//eventsMetrics.mu.Unlock()
		// }
	}
}

//func () updateStats
//eventsMetrics *EventsMetrics, respChan chan ReceivedEventsStats

//TODO const BUCKETS .005, .01, .025, .05, .075, .1, .25, .5, .75, 1.0, 2.5, 5.0, 7.5, 10.0

func ReceiverRun() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	fmt.Printf("Receiver starting ...\n")
	respChan := make(chan ReceivedEventsStats, 10*1000) // make size configurable based on concurrency?
	sleepSeconds := util.GetEnvFloat64("SLEEP_SECONDS", "0")
	log.Printf("SLEEP_SECONDS=%f", sleepSeconds)
	requestHandler := func(res http.ResponseWriter, req *http.Request) {
		contentType := req.Header.Get("Content-type")
		//fmt.Println(contentType)
		jsonMap := make(map[string]interface{})
		body, err := ioutil.ReadAll(req.Body)
		defer req.Body.Close()
		if err != nil {
			log.Printf("Error reading body: %v", err)
			http.Error(res, "can't read body", http.StatusBadRequest)
			return
		}
		if strings.Contains(contentType, "application/cloudevents+json") {
			err = json.Unmarshal(body, &jsonMap)
			if err != nil {
				log.Printf("Error unmarshaling body to JSON: %v", err)
				http.Error(res, err.Error(), 500)
				return
			}
		} else if strings.Contains(contentType, "application/json") {
			for name, values := range req.Header {
				for _, value := range values {
					//fmt.Println("header", name, value)
					if strings.HasPrefix(name, "Ce-") {
						key := strings.ToLower(strings.TrimPrefix(name, "Ce-"))
						//fmt.Println("json", key, value)
						jsonMap[key] = value
					}
				}
			}
			jsonData := make(map[string]interface{})
			err = json.Unmarshal(body, &jsonData)
			if err != nil {
				log.Printf("Error unmarshaling body to JSON: %v", err)
				http.Error(res, err.Error(), 500)
				return
			}
			jsonMap["data"] = jsonData
		} else {
			msg := "Content-Type header " + contentType + " is not supported"
			http.Error(res, msg, http.StatusUnsupportedMediaType)
			return
		}

		//log.Printf("Received JSON %s", jsonMap)

		//data := []byte("{'Hello' :  'World!'}")
		//res.Header().Set("Content-Type", "application/json")
		res.WriteHeader(http.StatusOK)
		//res.Write(data)
		//latencies := make([]float64, 1)
		//TODO calculate latency based on received timestamp
		//latencies[0] = 0.01
		labels := make(map[string]string)
		labelHeaders := [...]string{ //"source",
			"experimentid", "setupid", "workloadid"}
		for _, key := range labelHeaders {
			if val, found := jsonMap[key]; found {
				labels[key] = fmt.Sprint(val)
			}
		}
		experimentId := labels["experimentid"]
		setupId := labels["setupid"]
		workloadId := labels["workloadid"]
		//metricLabels := fmt.Sprintf("experimentId=%s,setupId=%s,workloadId=%s", experimentId, setupId, workloadId)
		source := fmt.Sprint(jsonMap["source"])
		eventId := fmt.Sprint(jsonMap["id"])
		timeStr := fmt.Sprint(jsonMap["time"])
		log.Printf("received event id %s from source %s (sleeping %f)", eventId, source, sleepSeconds)
		//stats := ReceivedEventsStats{&labels, source, eventId} //id, sequence, sequencemax, latencySeconds
		timestamp, err := time.Parse(time.RFC3339Nano, timeStr)
		if err != nil {
			log.Printf("Error unrecognized time format: %s %v", timeStr, err)
			http.Error(res, err.Error(), 500)
			return
		}
		latencySeconds := time.Since(timestamp).Seconds()
		stats := ReceivedEventsStats{experimentId, setupId, workloadId, source, eventId, latencySeconds}
		//&latencies}
		if sleepSeconds > 0 {
			milliseconds := sleepSeconds * 1000
			time.Sleep(time.Duration(milliseconds) * time.Millisecond)
		}

		respChan <- stats
	}

	if util.GetEnv("KAFKA_BOOTSTRAP_SERVERS", "") != "" {
		go KafkaReceive(respChan)
	}

	if util.GetEnv("REDIS_ADDRESS", "") != "" {
		go RedisReceive(respChan)
	}

	eventsMetrics := EventsMetrics{
		lock: &sync.RWMutex{},
		//events_total:                   0,
		events_total: make(map[string]*ExperimentStats),
		//events_latency_seconds_buckets: make(map[float64]int64, 16),
	}
	bucketList := util.GetEnv("BUCKET_LIST", "")
	eventsMetrics.events_total[""] = NewExperimentStats(events_metric, "", bucketList)

	//eventsMetrics.events_latency_seconds_buckets[0.005] = 1
	//var eventCount
	//eventCount := int64(0)
	metricsHandler := func(res http.ResponseWriter, req *http.Request) {
		res.Header().Set("Content-Type", "text/plain")
		res.WriteHeader(http.StatusOK)
		//eventsMetrics.mu.Lock()
		eventsMetrics.lock.RLock()
		defer eventsMetrics.lock.RUnlock()
		//eventsMetrics.mu.Unlock()
		//TODO better str ops
		var str string = ""
		//str = "events_total " + strconv.FormatInt(events_total[""], 10) + ".0\n"
		// for _, val := range eventsMetrics.events_total { // unsorted
		// 	//str = str + "events_total" + key + " " + strconv.FormatInt(val, 10) + ".0\n"
		// 	str += val.GetMetrics()
		// }
		sortedKeys := make([]string, 0, len(eventsMetrics.events_total))
		for key := range eventsMetrics.events_total {
			sortedKeys = append(sortedKeys, key)
		}
		sort.Strings(sortedKeys)
		for _, key := range sortedKeys {
			val := eventsMetrics.events_total[key]
			str += val.GetMetrics()
		}

		//value := eventsMetrics.events_latency_seconds_buckets[0.005]
		//str2 := str + "events_latency_seconds_bucket{le=\"0.01\"} " + strconv.FormatInt(value, 10) + ".0\n"
		data := []byte(str)
		res.Write(data)
	}

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	go aggegateMetrics(&eventsMetrics, respChan, ticker)

	// s := &http.Server{
	// 	Handler: requestHandler,
	// 	Name:    "Test Receiver",
	// }
	http.HandleFunc("/", requestHandler)
	http.HandleFunc("/metrics", metricsHandler)

	httpPort := util.GetEnv("PORT", "8001")
	//loc := "127.0.0.1:8001"
	loc := ":" + httpPort
	fmt.Printf("Starting receiver HTTP server on %s\n", loc)
	if err := http.ListenAndServe(loc, nil); err != nil {
		log.Fatalf("error in ListenAndServe: %s", err)
	}

}
