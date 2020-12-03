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
	servingv1client "knative.dev/serving/pkg/client/clientset/versioned/typed/serving/v1"
)

type Clean func(ksvcClient *servingv1client.ServingV1Client, ns, name string)

type BatchCleaner struct {
	namespaceNameList [][2]string
	concurrency       int
	ksvcClient        *servingv1client.ServingV1Client
	cleanFunc         Clean

	doneChan          chan bool
	namespaceNameChan chan [2]string
	finishedChan      chan int
	finishedCount     int
}

func NewBatchCleaner(namespaceNameList [][2]string, concurrency int, ksvcClient *servingv1client.ServingV1Client, cleanFunc Clean) *BatchCleaner {
	return &BatchCleaner{
		namespaceNameList: namespaceNameList,
		concurrency:       concurrency,
		namespaceNameChan: make(chan [2]string, len(namespaceNameList)),
		ksvcClient:        ksvcClient,
		cleanFunc:         cleanFunc,

		doneChan:      make(chan bool),
		finishedChan:  make(chan int, concurrency*5),
		finishedCount: 0,
	}
}

func (bc *BatchCleaner) Clean() {

	go bc.checkFinished()
	for i := 0; i < bc.concurrency; i++ {
		go bc.doClean()
	}
	for _, nsname := range bc.namespaceNameList {
		bc.namespaceNameChan <- nsname
	}
	<-bc.doneChan

}

func (bc *BatchCleaner) doClean() {
	for {
		select {
		case <-bc.doneChan:
			return
		case nsname := <-bc.namespaceNameChan:
			bc.cleanFunc(bc.ksvcClient, nsname[0], nsname[1])
			bc.finishedChan <- 1
		}
	}
}

func (bc *BatchCleaner) checkFinished() {
	for {
		select {
		case <-bc.finishedChan:
			bc.finishedCount++
			if bc.finishedCount >= len(bc.namespaceNameList) {
				close(bc.doneChan)
			}
		}
	}
}
