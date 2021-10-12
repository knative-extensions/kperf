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
	"strconv"
	"time"

	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/cloudevents/sdk-go/v2/types"
)

const eventType = "dev.knative.kperf.eventing.test"

type EventGenerator struct {
	plan            SendEventsPlan
	eventSeq        int
	eventsToSend    int
	eventsToSendStr string
}

func NewEventGenerator(plan SendEventsPlan) *EventGenerator {
	eg := EventGenerator{
		plan:         plan,
		eventSeq:     0,
		eventsToSend: int(float64(plan.eventsPerSecond) * plan.durationSeconds),
	}
	eg.eventsToSendStr = strconv.Itoa(eg.eventsToSend)
	return &eg
}

func (s *EventGenerator) EventRemainingToSend() int {
	return s.eventsToSend - s.eventSeq
}

func (s *EventGenerator) NextCloudEvents() []*event.Event {
	s.eventSeq++
	if s.eventSeq > s.eventsToSend {
		return nil
	}
	e := event.New("1.0")
	eventSeqStr := strconv.Itoa(s.eventSeq)
	e.SetID(eventSeqStr)
	e.SetSource(s.plan.senderName)
	e.SetType(eventType)
	e.SetTime(time.Now())
	e.SetExtension("experimentid", s.plan.experimentId)
	e.SetExtension("setupid", s.plan.setupId)
	e.SetExtension("workloadid", s.plan.workloadId)
	e.SetExtension("sequence", eventSeqStr)
	e.SetExtension("sequencetype", "Integer")
	e.SetExtension("sequencemax", s.eventsToSendStr)
	data := []byte("{}")
	e.SetData(event.ApplicationJSON, data)
	events := make([]*event.Event, 1)
	events[0] = &e
	return events
}

func (s *EventGenerator) NextCloudEventsAsMaps() []*map[string]string {
	s.eventSeq++
	if s.eventSeq > s.eventsToSend {
		return nil
	}
	eventSeqStr := strconv.Itoa(s.eventSeq)
	timestamp := types.Timestamp{Time: time.Now()}
	timestampStr := timestamp.String()
	values := map[string]string{"specversion": "1.0", "id": eventSeqStr, "source": s.plan.senderName, "type": eventType, "time": timestampStr, "experimentid": s.plan.experimentId, "setupid": s.plan.setupId, "workloadid": s.plan.workloadId, "sequence": eventSeqStr, "sequencetype": "Integer", "sequencemax": s.eventsToSendStr, "data": "{}"}
	events := make([]*map[string]string, 1)
	events[0] = &values
	return events
}
