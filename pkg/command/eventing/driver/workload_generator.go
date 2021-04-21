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

import "github.com/cloudevents/sdk-go/v2/event"

type EventGenerator struct {
	plan     SendEventsPlan
	eventSeq int
}

func NewEventGenerator(plan SendEventsPlan) EventGenerator {
	eg := EventGenerator{
		plan:     plan,
		eventSeq: 0,
	}
	return eg
}

func (s EventGenerator) NextCloudEvents() []*event.Event {
	e := event.Event{}
	events := make([]*event.Event, 1, 100)
	events[0] = &e
	return events
}

func (s EventGenerator) NextCloudEventsAsMaps() []*map[string]string {
	events := make([]*map[string]string, 1, 100)
	values := map[string]string{"id": "1234668888888", "source": "323223232332909090", "type": "dev.knative.eventing.test.scaling", "timestamp": "12929299999992222"}
	s.eventSeq++
	events[0] = &values
	return events
}
