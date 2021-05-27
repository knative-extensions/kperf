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
package service

import (
	"sync"
	"time"
)

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

	verbose    bool
	checkReady bool
	timeout    time.Duration
}

type generateResult struct {
	ksvcGenerateSuccess int
	ksvcGenerateFail    int
	ksvcReady           int
	ksvcNotReady        int
	m                   sync.Mutex
}

func (g *generateResult) SetSuccess() {
	g.m.Lock()
	defer g.m.Unlock()
	g.ksvcGenerateSuccess += 1
}

func (g *generateResult) SetFail() {
	g.m.Lock()
	defer g.m.Unlock()
	g.ksvcGenerateFail += 1
}

type cleanArgs struct {
	namespacePrefix string
	namespaceRange  string
	namespace       string
	svcPrefix       string
	concurrency     int
	verbose         bool
}

type cleanResult struct {
	ksvcCleanSuccess int
	ksvcCleanFail    int
	m                sync.Mutex
}

func (c *cleanResult) SetSuccess() {
	c.m.Lock()
	defer c.m.Unlock()
	c.ksvcCleanSuccess += 1
}

func (c *cleanResult) SetFail() {
	c.m.Lock()
	defer c.m.Unlock()
	c.ksvcCleanFail += 1
}

type measureArgs struct {
	svcRange        string
	namespace       string
	svcPrefix       string
	namespaceRange  string
	namespacePrefix string
	concurrency     int
	verbose         bool
	output          string
}

type measureResult struct {
	Sums         sums `json:"-"`
	Result       result
	Service      serviceCount
	KnativeInfo  knativeInfo
	SvcReadyTime []float64 `json:"-"`
}

type sums struct {
	svcConfigurationsReadySum         float64
	svcRoutesReadySum                 float64
	svcReadySum                       float64
	revisionReadySum                  float64
	kpaActiveSum                      float64
	sksReadySum                       float64
	sksActivatorEndpointsPopulatedSum float64
	sksEndpointsPopulatedSum          float64
	ingressReadySum                   float64
	ingressNetworkConfiguredSum       float64
	ingressLoadBalancerReadySum       float64
	podScheduledSum                   float64
	containersReadySum                float64
	queueProxyStartedSum              float64
	userContrainerStartedSum          float64
	deploymentCreatedSum              float64
}

type serviceCount struct {
	ReadyCount    int `json:"Ready"`
	NotReadyCount int `json:"NotReady"`
	NotFoundCount int `json:"NotFound"`
	FailCount     int `json:"Fail"`
}

type knativeInfo struct {
	ServingVersion    string
	EventingVersion   string
	IngressController string
	IngressVersion    string
}

type result struct {
	AverageSvcConfigurationReadySum          float64 `json:"AverageConfigurationDuration"`
	AverageRevisionReadySum                  float64 `json:"AverageRevisionDuration"`
	AverageDeploymentCreatedSum              float64 `json:"AverageDeploymentDuration"`
	AveragePodScheduledSum                   float64 `json:"AveragePodScheduleDuration"`
	AverageContainersReadySum                float64 `json:"AveragePodContainersReadyDuration"`
	AverageQueueProxyStartedSum              float64 `json:"AveragePodQueueProxyStartedDuration"`
	AverageUserContrainerStartedSum          float64 `json:"AveragePodUserContainerStartedDuration"`
	AverageKpaActiveSum                      float64 `json:"AverageAutoscalerActiveDuration"`
	AverageSksReadySum                       float64 `json:"AverageServiceReadyDuration"`
	AverageSksActivatorEndpointsPopulatedSum float64 `json:"AverageServiceActivatorEndpointsPopulatedDuration"`
	AverageSksEndpointsPopulatedSum          float64 `json:"AverageServiceEndpointsPopulatedDuration"`
	AverageSvcRoutesReadySum                 float64 `json:"AverageServiceRouteReadyDuration"`
	AverageIngressReadySum                   float64 `json:"AverageIngressReadyDuration"`
	AverageIngressNetworkConfiguredSum       float64 `json:"AverageIngressNetworkConfiguredDuration"`
	AverageIngressLoadBalancerReadySum       float64 `json:"AverageIngressLoadBalancerReadyDuration"`
	OverallTotal                             float64 `json:"Total"`
	OverallAverage                           float64 `json:"Average"`
	OverallMedian                            float64 `json:"Median"`
	OverallMin                               float64 `json:"Min"`
	OverallMax                               float64 `json:"Max"`
	P50                                      float64 `json:"Percentile50"`
	P90                                      float64 `json:"Percentile90"`
	P95                                      float64 `json:"Percentile95"`
	P98                                      float64 `json:"Percentile98"`
	P99                                      float64 `json:"Percentile99"`
}
