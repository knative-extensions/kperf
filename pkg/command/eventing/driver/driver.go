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
	"fmt"
	"log"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"knative.dev/kperf/pkg/command/eventing/util"
)

type SendEventsPlan struct {
	senderName      string
	eventsPerSecond int
	durationSeconds float64
	targetUrl       string
}

type EventsStats struct {
	senderName      string
	eventsSent      int
	durationSeconds float64
	size            int64
	errors          int
}

type EventSender interface {
	Send() EventsStats
}

func doSending(sender EventSender, respChan chan EventsStats) {
	stats := sender.Send()
	respChan <- stats
}

type TestConfig struct {
	targetUrl           string
	runId               string
	concurrent          int
	start               int
	durationSeconds     float64
	inc                 int
	testDurationSeconds float64
}

func readConfig() TestConfig {
	var config TestConfig
	config.targetUrl = util.GetEnv("TARGET_URL", "http://localhost:8001")
	uuid := uuid.New()
	config.runId = util.GetEnv("RUN_ID", uuid.String())
	config.concurrent = util.GetEnvInt("CONCURRENT", "1")
	config.start = util.GetEnvInt("START", "500")
	config.inc = util.GetEnvInt("INC", "500")
	config.durationSeconds = util.GetEnvFloat64("DURATION", "0.01")
	config.testDurationSeconds = util.GetEnvFloat64("TEST_DURATION", "0.02")
	return config
}

func senderForWorkloadPlan(plan SendEventsPlan, http *http.Transport) EventSender {
	var sender EventSender
	if strings.HasPrefix(plan.targetUrl, "http") {
		sender = HttpEventSender{plan, http}
	} else if strings.HasPrefix(plan.targetUrl, "kafka") {
		sender = KafkaEventSender{plan}
	} else if strings.HasPrefix(plan.targetUrl, "rediss") {
		sender = NewRedisSender(plan)
	} else {
		log.Fatal("unknon target to send event ", plan.targetUrl)
	}
	return sender
}

func DriveWorkload() {
	config := readConfig()
	//util.UtilMe()
	runtime.GOMAXPROCS(runtime.NumCPU())
	startTime := time.Now()
	respChan := make(chan EventsStats)
	http := &http.Transport{}
	eventsToSend := config.start
	durationSeconds := config.durationSeconds
	testDurationSeconds := config.testDurationSeconds
	targetEndTime := startTime.Add(time.Duration(testDurationSeconds) * time.Second)
	eventsCount := 0
	errorsCount := 0
	phaseCounter := 0
	for {
		phaseCounter++
		phaseStartTime := time.Now()
		phaseName := "phase-" + strconv.Itoa(phaseCounter)
		for i := 0; i < config.concurrent; i++ {
			name := "test-sender-" + strconv.Itoa(i+1)
			plan := SendEventsPlan{name, eventsToSend, durationSeconds, config.targetUrl}
			//sender := FakeEventSender{plan, 0.001}
			sender := senderForWorkloadPlan(plan, http)
			go doSending(sender, respChan)
		}
		//get results
		phaseEventCount := 0
		phaseErrorsCount := 0
		for i := 0; i < config.concurrent; i++ {
			r, ok := <-respChan
			if ok {
				phaseEventCount += r.eventsSent
				eventsCount += r.eventsSent
				phaseErrorsCount += r.errors
				errorsCount += r.errors
				fmt.Printf("stats %+v\n", r)
			}
		}
		endTime := time.Now()
		// print stats for this phase
		phaseDuration := endTime.Sub(phaseStartTime)
		phaseTimeSeconds := float64(phaseDuration.Nanoseconds()) / float64(time.Second)
		phaseEventsPerSecond := float64(phaseEventCount) / phaseTimeSeconds
		fmt.Printf("Phase %s took %s to send %d events reaching %f [events/second] (erros %d)\n", phaseName, phaseDuration, phaseEventCount, phaseEventsPerSecond, phaseErrorsCount)
		eventsToSend += config.inc
		if endTime.After(targetEndTime) {
			break
		}
	}

	duration := time.Since(startTime)
	ns := duration.Nanoseconds()
	timeSeconds := float64(duration.Nanoseconds()) / float64(time.Second)
	eventsPerSecond := float64(eventsCount) / timeSeconds

	fmt.Printf("Took %s (%f [s] %d [ns]) to send %d events reaching %f [events/second] (errors %d)\n", duration, timeSeconds, ns, eventsCount, eventsPerSecond, errorsCount)
}
