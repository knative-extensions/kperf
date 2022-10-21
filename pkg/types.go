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
package pkg

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	networkingv1alpha1 "knative.dev/networking/pkg/client/clientset/versioned/typed/networking/v1alpha1"
	autoscalingv1alpha1 "knative.dev/serving/pkg/client/clientset/versioned/typed/autoscaling/v1alpha1"
	servingv1client "knative.dev/serving/pkg/client/clientset/versioned/typed/serving/v1"
)

type PerfParams struct {
	KubeCfgPath          string
	ClientConfig         clientcmd.ClientConfig
	ClientSet            kubernetes.Interface
	NewAutoscalingClient func() (autoscalingv1alpha1.AutoscalingV1alpha1Interface, error)
	NewServingClient     func() (servingv1client.ServingV1Interface, error)
	NewNetworkingClient  func() (networkingv1alpha1.NetworkingV1alpha1Interface, error)
}

type GenerateArgs struct {
	Number      int
	Interval    int
	Batch       int
	Concurrency int
	MinScale    int
	MaxScale    int

	NamespacePrefix string
	NamespaceRange  string
	Namespace       string
	SvcPrefix       string

	CheckReady bool
	Timeout    time.Duration

	Template string
}

type CleanArgs struct {
	NamespacePrefix string
	NamespaceRange  string
	Namespace       string
	SvcPrefix       string
	Concurrency     int
}

type MeasureArgs struct {
	SvcRange        string
	Namespace       string
	SvcPrefix       string
	NamespaceRange  string
	NamespacePrefix string
	Concurrency     int
	Verbose         bool
	Output          string
}

type ScaleArgs struct {
	Svc              string
	SvcRange         string
	Namespace        string
	SvcPrefix        string
	NamespaceRange   string
	NamespacePrefix  string
	Concurrency      int
	MaxRetries       int
	RequestInterval  time.Duration
	RequestTimeout   time.Duration
	ResolvableDomain bool
	Verbose          bool
	Output           string
	Https            bool
}

type LoadArgs struct {
	Svc                   string
	SvcRange              string
	Namespace             string
	SvcPrefix             string
	NamespaceRange        string
	NamespacePrefix       string
	Verbose               bool
	ResolvableDomain      bool
	WaitPodsReadyDuration time.Duration
	Output                string
	LoadTool              string
	LoadDuration          string
	LoadConcurrency       string
	Https                 bool
}

type MeasureResult struct {
	Sums         Sums `json:"-"`
	Result       Result
	Service      ServiceCount
	KnativeInfo  KnativeInfo
	SvcReadyTime []float64 `json:"-"`
}

type ScaleResult struct {
	KnativeInfo KnativeInfo
	Measurment  []ScaleFromZeroResult
}

type ScaleFromZeroResult struct {
	ServiceName       string
	ServiceNamespace  string
	ServiceLatency    float64 `json:"serviceLatency"`
	DeploymentLatency float64 `json:"deploymentLatency"`
}

type LoadResult struct {
	KnativeInfo KnativeInfo
	Measurment  []LoadFromZeroResult
}

type LoadFromZeroResult struct {
	ServiceName        string
	ServiceNamespace   string
	TotalReadyReplicas int
	TotalReadyPods     int
	ReplicaResults     []LoadReplicaResult
	PodResults         []LoadPodResult
}

type LoadReplicaResult struct {
	ReadyReplicasCount   int
	ReplicaReadyTime     time.Time
	ReplicaReadyDuration float64
}
type LoadPodResult struct {
	PodCreateTime    metav1.Time
	PodReadyTime     metav1.Time
	PodReadyDuration float64
}
type Sums struct {
	SvcConfigurationsReadySum         float64
	SvcRoutesReadySum                 float64
	SvcReadySum                       float64
	RevisionReadySum                  float64
	KpaActiveSum                      float64
	SksReadySum                       float64
	SksActivatorEndpointsPopulatedSum float64
	SksEndpointsPopulatedSum          float64
	IngressReadySum                   float64
	IngressNetworkConfiguredSum       float64
	IngressLoadBalancerReadySum       float64
	PodScheduledSum                   float64
	ContainersReadySum                float64
	QueueProxyStartedSum              float64
	UserContrainerStartedSum          float64
	DeploymentCreatedSum              float64
}

type ServiceCount struct {
	ReadyCount    int `json:"Ready"`
	NotReadyCount int `json:"NotReady"`
	NotFoundCount int `json:"NotFound"`
	FailCount     int `json:"Fail"`
}

type KnativeInfo struct {
	ServingVersion    string
	EventingVersion   string
	IngressController string
	IngressVersion    string
}

type Result struct {
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
