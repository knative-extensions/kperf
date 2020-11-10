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
	"sync"
	"time"
)

var (
	count, interval, batch, concurrency, minScale, maxScale int
	nsPrefix, nsRange, ns                                   string
	svcPrefix, svcRange, svcName                            string
	svcNamePrefixDefault                                    string = "testksvc"
	checkReady                                              bool
	timeout                                                 time.Duration
	svcConfigurationsReadySum, svcRoutesReadyReadySum, svcReadySum, minDomainReadySum, maxDomainReadySum,
	revisionReadySum, podAutoscalerReadySum, ingressReadyReadySum, ingressNetworkConfiguredSum,
	ingressLoadBalancerReadySum, podScheduledSum, containersReadySum, queueProxyStartedSum,
	userContrainerStartedSum, deploymentCreatedSum float64
	readyCount, notReadyCount, notFoundCount, measureJob int
	lock                                                 sync.Mutex
	err                                                  error
	verbose                                              bool
)
