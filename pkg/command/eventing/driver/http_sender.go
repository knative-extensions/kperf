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
	"time"
)

type HttpEventSender struct {
	Plan SendEventsPlan
	http *http.Transport
}

func (s HttpEventSender) Send() EventsStats {
	plan := s.Plan
	g := NewEventGenerator(plan)
	senderName := plan.senderName
	startTime := time.Now()
	targetEndTime := startTime.Add(time.Duration(plan.durationSeconds) * time.Second)
	errCount := 0
	eventsSentCount := 0
	var contentLenght int64 = 0
	//TODO divide events to send into chunks/batches for perf impact
	chunkSize := 1
	chunkCount := 0
	//TODO test HTTP 2 pipelining
	//TODO test CloudEvents batch
	for {
		//TODO switch between different types of events
		events := g.NextCloudEventsAsMaps()
		if events == nil {
			break
		}
		for _, eventMap := range events {
			event, err := json.Marshal(eventMap)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Printf("Sending to %s event %s\n", plan.targetUrl, event)
			req, err := http.NewRequest("POST", plan.targetUrl, bytes.NewBuffer(event))
			if err != nil {
				log.Println(err)
			}
			req.Header.Set("Content-Type", "application/cloudevents+json; charset=UTF-8")
			resp, err := s.http.RoundTrip(req)
			// TODO check response HTTP code?
			// count successes and errors
			if err != nil {
				errCount++
			} else {
				eventsSentCount++
				contentLenght += resp.ContentLength
			}
			chunkCount++
			if chunkCount >= chunkSize {
				chunkCount = 0
				// sleep to keep events/second goal
				//durationSoFar := time.Since(start)
			}
			eventsToSend := g.EventRemainingToSend()
			soFarTime := time.Now()
			if eventsToSend > 0 && soFarTime.Before(targetEndTime) {
				remainingDuration := targetEndTime.Sub(soFarTime)
				sleepDuration := remainingDuration / time.Duration(eventsToSend)
				//fmt.Printf("sending sleeping %s\n", sleepDuration)
				time.Sleep(sleepDuration)

			}

		}
	}
	// sleep for remaining time to reach events/second
	endTime := time.Now()
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
