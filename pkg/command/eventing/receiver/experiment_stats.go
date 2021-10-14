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
	"strconv"
	"strings"
)

type ExperimentStats struct {
	name          string
	labels        string
	commaLabels   string
	bucketList    string
	boundaries    []float64
	boundariesStr []string
	buckets       []int64
	count         int64
	sum           float64
}

func ParseBucketsBoundaries(commaSeparatedlistOfBoundaries string) ([]float64, []string, error) {
	bucketListTrimmed := strings.TrimSpace(commaSeparatedlistOfBoundaries)
	bucketsSlice := strings.Split(commaSeparatedlistOfBoundaries, ",")
	lenBoundaries := len(bucketsSlice)
	if len(bucketListTrimmed) == 0 {
		lenBoundaries = 0
	}
	boundaries := make([]float64, lenBoundaries)
	boundariesStr := make([]string, lenBoundaries)
	//fmt.Println(len(bucketsSlice))
	if lenBoundaries > 0 {
		//TODO: check values are positive and in increasing order
		for pos, item := range bucketsSlice {
			//fmt.Println(item)
			itemTrimmed := strings.TrimSpace(item)
			val, err := strconv.ParseFloat(itemTrimmed, 64)
			if err != nil {
				return nil, nil, err
			}
			//fmt.Printf("%f\n", val)
			boundaries[pos] = val
			boundariesStr[pos] = itemTrimmed
		}
	}
	return boundaries, boundariesStr, nil
}

func NewExperimentStats(name string, labels string, bucketList string) *ExperimentStats {
	boundaries, boundariesStr, err := ParseBucketsBoundaries(bucketList)
	if err != nil {
		panic(err)
	}
	h := ExperimentStats{
		name:          name,
		labels:        labels,
		bucketList:    bucketList,
		boundaries:    boundaries,
		boundariesStr: boundariesStr,
	}
	if len(h.labels) > 0 {
		h.commaLabels = "," + h.labels
	}
	h.buckets = make([]int64, len(boundaries))
	return &h
}

func (h *ExperimentStats) GetBucketList() string {
	return h.bucketList
}

func (h *ExperimentStats) Add(val float64) {
	for pos, item := range h.boundaries {
		//fmt.Println("val=", val, " pos=", pos, " item=", item)
		if val <= item {
			h.buckets[pos]++
		}
	}
	h.sum = h.sum + val
	h.count++
}

func (h *ExperimentStats) GetMetrics() string {
	var str string = ""
	if len(h.buckets) > 0 {
		str += "# TYPE " + h.name + " histogram\n"
	}
	for key, val := range h.buckets {
		//boundary := h.boundaries[key]
		boundaryStr := h.boundariesStr[key] //fmt.Sprintf("%f", boundary)
		str = str + h.name + "_bucket{le=\"" + boundaryStr + "\"" + h.commaLabels + "} " + strconv.FormatInt(val, 10) + ".0\n"
	}
	if len(h.buckets) > 0 {
		str = str + h.name + "_bucket{le=\"+Inf\"" + h.commaLabels + "} " + strconv.FormatInt(h.count, 10) + ".0\n"
	}
	labels := ""
	if len(h.commaLabels) > 0 {
		labels = "{" + h.labels + "}"
	}
	if len(h.buckets) > 0 {
		str = str + h.name + "_count" + labels + " " + strconv.FormatInt(h.count, 10) + ".0\n"
	}

	str += "# TYPE " + h.name + " counter\n"
	str = str + h.name + "_total" + labels + " " + strconv.FormatInt(h.count, 10) + ".0\n"

	return str
}

func (h *ExperimentStats) Merge(h2 *ExperimentStats) {
	if len(h.buckets) != len(h2.buckets) {
		panic("bukets len mismatch") //should never happen
	}
	for pos, item := range h2.buckets {
		h.buckets[pos] += item
	}
	h.count += h2.count
	h.sum += h2.sum
}
