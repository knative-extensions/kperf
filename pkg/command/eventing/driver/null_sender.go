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

import "time"

type NullEventSender struct {
	Plan             SendEventsPlan
	SleepTimeSeconds float64 // simulate sending time
}

func (s NullEventSender) Send() EventsStats {
	plan := s.Plan
	eventsSent := int(float64(plan.eventsPerSecond) * plan.durationSeconds)
	// TODO better golang idiom/conversion?
	sleepTimeNano := int64(plan.durationSeconds * float64(time.Second))
	time.Sleep(time.Duration(sleepTimeNano))
	timeSeconds := plan.durationSeconds
	stats := EventsStats{plan.senderName, eventsSent, timeSeconds, 0, 0}
	return stats
}
