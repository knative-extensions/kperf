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

package driver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
)

type HttpEventSender struct {
	Plan SendEventsPlan
	http *http.Transport
}

func (s HttpEventSender) Send() EventsStats {
	plan := s.Plan
	senderName := plan.senderName
	values := map[string]string{"id": "1234668888888", "source": "323223232332909090", "type": "dev.knative.eventing.test.scaling", "timestamp": "12929299999992222"}
	startTime := time.Now()
	errCount := 0
	eventsSentCount := 0
	var contentLenght int64 = 0
	//TODO divide events to send into chunks/batches for perf impact
	chunkSize := 1
	chunkCount := 0
	eventsToSend := int(float64(plan.eventsPerSecond) * plan.durationSeconds)
	//TODO test HTTP 2 pipelining
	//TODO test CLoudEvents batch
	for i := 0; i < eventsToSend; i++ {
		values["id"] = strconv.Itoa(i + 1)
		event, err := json.Marshal(values)
		if err != nil {
			log.Fatal(err)
		}
		req, err := http.NewRequest("POST", plan.targetUrl, bytes.NewBuffer(event))
		if err != nil {
			log.Println(err)
		}
		resp, err := s.http.RoundTrip(req)
		eventsSentCount++
		// TODO check response HTTP code?
		// count successes and errors
		if err != nil {
			errCount++
		} else {
			contentLenght += resp.ContentLength
		}
		chunkCount++
		if chunkCount >= chunkSize {
			chunkCount = 0
			// sleep to keep events/second goal
			//durationSoFar := time.Since(start)

		}
	}
	// sleep for remaining time to reach events/second
	endTime := time.Now()
	targetEndTime := startTime.Add(time.Duration(plan.durationSeconds) * time.Second)
	//duration := time.Since(start)
	duration := endTime.Sub(startTime)
	if endTime.Before(targetEndTime) {
		sleepDuration := targetEndTime.Sub(endTime)
		fmt.Printf("Sender %s sleeping %s\n", senderName, sleepDuration)
		time.Sleep(sleepDuration)
	}
	timeSeconds := float64(duration.Nanoseconds()) / float64(time.Second)
	stats := EventsStats{plan.senderName, eventsSentCount, timeSeconds, contentLenght, errCount}
	return stats
}
