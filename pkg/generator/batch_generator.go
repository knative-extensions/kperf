// Copyright © 2020 The Knative Authors
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

package generator

import (
	"os"
	"time"

	"knative.dev/kperf/pkg"
)

// func Generator do the generate action in namespace ns with the index as the suffix of the resource name
// it returns the name and namespace of the generated resource
type Generator func(ns string, index int) (string, string)

// func PostGenerator is executed after Generator has been executed.
// if the error is not nil, the whole generate process will exit
type PostGenerator func(string, string) error

// BatchGenerator helps generate `count` of resource with `concurrency` number of gorutines. It executes `generateFunc` `batch` times
// per time, and between each batch, it stop for `interval` time.
type BatchGenerator struct {
	interval          time.Duration
	count             int
	counter           int
	batch             int
	concurrency       int
	namespaceList     []string
	generateFunc      Generator
	postGeneratorFunc PostGenerator
	params            *pkg.PerfParams

	indexChan     chan int
	finishedChan  chan int
	finishedCount int
	doneChan      chan bool
}

func NewBatchGenerator(interval time.Duration, count, batch int, concurrency int, namespaceList []string, generator Generator, postGenerator PostGenerator) *BatchGenerator {
	return &BatchGenerator{
		interval:          interval,
		count:             count,
		counter:           0,
		batch:             batch,
		concurrency:       concurrency,
		namespaceList:     namespaceList,
		generateFunc:      generator,
		postGeneratorFunc: postGenerator,

		indexChan:     make(chan int, batch*5),
		finishedChan:  make(chan int, batch*5),
		finishedCount: 0,
		doneChan:      make(chan bool),
	}
}

func (bg *BatchGenerator) Generate() {
	// avoid the blocked channel
	if bg.count == 0 {
		return
	}
	ticker := time.NewTicker(bg.interval)
	defer ticker.Stop()
	go bg.checkFinished()
	for i := 0; i < bg.concurrency; i++ {
		go bg.doGenerate()
	}
	for {
		select {
		case <-bg.doneChan:
			return
		case <-ticker.C:
			i := 0
			for bg.counter < bg.count && i < bg.batch {
				bg.indexChan <- bg.counter
				bg.counter++
				i++
			}
		}
	}

}

func (bg *BatchGenerator) doGenerate() {
	for {
		select {
		case <-bg.doneChan:
			return
		case index := <-bg.indexChan:
			ns := bg.namespaceList[index%len(bg.namespaceList)]
			ns, name := bg.generateFunc(ns, index)
			if bg.postGeneratorFunc(ns, name) != nil {
				os.Exit(1)
			}
			bg.finishedChan <- 1
		}
	}
}

func (bg *BatchGenerator) checkFinished() {
	for {
		select {
		case <-bg.finishedChan:
			bg.finishedCount++
			if bg.finishedCount >= bg.count {
				close(bg.doneChan)
			}
		}
	}
}
