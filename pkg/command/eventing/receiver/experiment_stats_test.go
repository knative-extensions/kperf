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
	"testing"
	"time"
)

func TestLatencyCalculations(t *testing.T) {
	//Ce-Time 2021-07-27T22:32:14.591468983Z
	timestamp, err := time.Parse(time.RFC3339Nano, "2021-07-27T22:32:14.591468983Z")
	if err != nil {
		panic(err)
	}
	fmt.Println(timestamp)

	tim2 := timestamp.Add(time.Millisecond * 123)

	secs := tim2.Sub(timestamp).Seconds()
	fmt.Println(secs)

	//secs := time.Since(tim).Seconds()
	latencySecnds := time.Since(timestamp).Seconds()
	fmt.Println("latencySecnds=", latencySecnds)

	// parse buckets
	testLatencyCalculationsForBucketList(t, "")
	bucketList := "0.005, 0.01, 0.1, 1.0, 2.5, 5.0, 7.5, 10.0"
	// bucketsSlice := strings.Split(bucketList, ",")
	testLatencyCalculationsForBucketList(t, bucketList)
}

func testLatencyCalculationsForBucketList(t *testing.T, bucketList string) {
	// boundaries := make([]float64, len(bucketsSlice))
	// for pos, item := range bucketsSlice {
	// 	fmt.Println(item)
	// 	itemTrimmed := strings.TrimSpace(item)
	// 	val, err := strconv.ParseFloat(itemTrimmed, 64)
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	fmt.Printf("%f\n", val)
	// 	boundaries[pos] = val
	// }
	// fmt.Println(boundaries)

	// create Histogram (name : str, labels : str, boundaries []float)
	h := NewExperimentStats("events", "env=\"Test\"", bucketList)
	//add(latencySeconds float)
	h.Add(0.0001)
	h.Add(0.02)
	h.Add(1)
	h.Add(12)
	//printMetrics
	fmt.Println(h.GetMetrics())
	//h.AddHistogram(h2)

	hAll := NewExperimentStats("events", "", bucketList)
	hAll.Merge(h)
	fmt.Println(hAll.GetMetrics())
}
