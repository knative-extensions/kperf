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
package service

import (
	"time"
)

const svcNamePrefixDefault string = "testksvc"

type generateArgs struct {
	number      int
	interval    int
	batch       int
	concurrency int
	minScale    int
	maxScale    int

	namespacePrefix string
	namespaceRange  string
	namespace       string
	svcPrefix       string
	svcRange        string
	svcName         string

	checkReady bool
	timeout    time.Duration
}

type cleanArgs struct {
	namespacePrefix string
	namespaceRange  string
	namespace       string
	svcPrefix       string
	concurrency     int
}
type measureArgs struct {
	svcRange        string
	namespace       string
	svcPrefix       string
	namespaceRange  string
	namespacePrefix string
	concurrency     int
	verbose         bool
}
type measureResult struct {
	svcConfigurationsReadySum   float64
	svcRoutesReadySum           float64
	svcReadySum                 float64
	minDomainReadySum           float64
	maxDomainReadySum           float64
	revisionReadySum            float64
	podAutoscalerReadySum       float64
	ingressReadySum             float64
	ingressNetworkConfiguredSum float64
	ingressLoadBalancerReadySum float64
	podScheduledSum             float64
	containersReadySum          float64
	queueProxyStartedSum        float64
	userContrainerStartedSum    float64
	deploymentCreatedSum        float64

	readyCount    int
	notReadyCount int
	notFoundCount int

	svcReadyTime []float64
}
