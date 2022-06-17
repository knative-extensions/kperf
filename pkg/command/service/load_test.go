// Copyright 2022 The Knative Authors
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
	"context"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"strconv"
	"testing"
	"time"

	"bou.ke/monkey"
	"gotest.tools/v3/assert"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"
	"knative.dev/kperf/pkg"
	"knative.dev/kperf/pkg/testutil"
	networkingv1alpha1 "knative.dev/networking/pkg/client/clientset/versioned/typed/networking/v1alpha1"
	fakenetworkingv1alpha1 "knative.dev/networking/pkg/client/clientset/versioned/typed/networking/v1alpha1/fake"
	"knative.dev/pkg/apis"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	autoscalingv1client "knative.dev/serving/pkg/client/clientset/versioned/typed/autoscaling/v1alpha1"
	autoscalingv1fake "knative.dev/serving/pkg/client/clientset/versioned/typed/autoscaling/v1alpha1/fake"
	servingv1client "knative.dev/serving/pkg/client/clientset/versioned/typed/serving/v1"
	servingv1fake "knative.dev/serving/pkg/client/clientset/versioned/typed/serving/v1/fake"
)

const (
	FakeHostIP             = "192.168.0.1"
	FakeNamespace          = "kest"
	FakeServicePrefix      = "ktest"
	FakeServiceName        = "ktest-0"
	FakeNodePort           = "32283"
	FakeIngressServiceName = "istio-ingressgateway"
	FakeIngressNamespace   = "istio-system"
	FakeEndpoint           = "http://192.168.0.1:32283"
	FakeLoadConcurrency    = "30"
	FakeLoadDuration       = "60s"
)

func TestNewServiceLoadCommand(t *testing.T) {
	// pre-test
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: FakeNamespace,
		},
	}
	client := k8sfake.NewSimpleClientset(ns)
	fakeAutoscaling := &autoscalingv1fake.FakeAutoscalingV1alpha1{Fake: &clienttesting.Fake{}}
	autoscalingClient := func() (autoscalingv1client.AutoscalingV1alpha1Interface, error) {
		return fakeAutoscaling, nil
	}

	fakeServing := &servingv1fake.FakeServingV1{Fake: &clienttesting.Fake{}}
	servingClient := func() (servingv1client.ServingV1Interface, error) {
		return fakeServing, nil
	}

	fakeNetworking := &fakenetworkingv1alpha1.FakeNetworkingV1alpha1{Fake: &clienttesting.Fake{}}
	networkingClient := func() (networkingv1alpha1.NetworkingV1alpha1Interface, error) {
		return fakeNetworking, nil
	}

	p := &pkg.PerfParams{
		ClientSet:            client,
		NewAutoscalingClient: autoscalingClient,
		NewServingClient:     servingClient,
		NewNetworkingClient:  networkingClient,
	}

	t.Run("uncompleted or wrong args for service load", func(t *testing.T) {
		cmd := NewServiceLoadCommand(p)
		_, err := testutil.ExecuteCommand(cmd)
		assert.ErrorContains(t, err, "'service load' requires flag(s)")

		_, err = testutil.ExecuteCommand(cmd, "--namespace-prefix", FakeNamespace, "--namespace-range", "1200")
		assert.ErrorContains(t, err, "expected range like 1,500, given 1200")

		_, err = testutil.ExecuteCommand(cmd, "--namespace-prefix", "ns-1", "--namespace-range", "1,2")
		assert.ErrorContains(t, err, "no namespace found with prefix ns-1")

		_, err = testutil.ExecuteCommand(cmd, "--namespace-range", "x,y", "--namespace", FakeNamespace)
		assert.ErrorContains(t, err, "strconv.Atoi: parsing \"x\": invalid syntax")

		_, err = testutil.ExecuteCommand(cmd, "--namespace-range", "1,y", "--namespace", FakeNamespace)
		assert.ErrorContains(t, err, "strconv.Atoi: parsing \"y\": invalid syntax")

		_, err = testutil.ExecuteCommand(cmd, "--namespace-range", "1,0", "--namespace-prefix", FakeNamespace)
		assert.ErrorContains(t, err, "failed to parse namespace range 1,0")

	})
	t.Run("both namespace and namespace-prefix are empty", func(t *testing.T) {
		cmd := NewServiceLoadCommand(p)
		_, err := testutil.ExecuteCommand(cmd, "--namespace-range", "1,0", "namespacePrefix", "")
		assert.ErrorContains(t, err, "both namespace and namespace-prefix are empty")
	})
	t.Run("load service as expected with error namespace range", func(t *testing.T) {
		cmd := NewServiceLoadCommand(p)
		_, err := testutil.ExecuteCommand(cmd, "--svc-prefix", "svc", "--namespace-prefix", "ns", "--namespace-range", "1,1")
		assert.ErrorContains(t, err, "no namespace found with prefix ns")
	})
	t.Run("load service success", func(t *testing.T) {
		defer func() {
			deleteWrkCmd := "rm -rf wrk*.lua"
			runDeleteWrkCmd := exec.Command("/bin/sh", "-c", deleteWrkCmd)
			_, err := runDeleteWrkCmd.Output()
			if err != nil {
				fmt.Printf("delete wrk lua command error\n")
			}
			deleteOutputCmd := "rm ./*" + LoadOutputFilename + "*"
			runDeleteOutputCmd := exec.Command("/bin/sh", "-c", deleteOutputCmd)
			_, err = runDeleteOutputCmd.Output()
			if err != nil {
				fmt.Printf("delete output command error\n")
			}
		}()

		cmd := NewServiceLoadCommand(p)
		_, err := testutil.ExecuteCommand(cmd, "--svc-prefix", FakeServicePrefix, "--namespace", FakeNamespace, "--range", "0,0", "--load-tool", "hey", "--load-concurrency", FakeLoadConcurrency, "--load-duration", FakeLoadDuration, "--output", "./")
		assert.NilError(t, err)

		_, err = testutil.ExecuteCommand(cmd, "--svc-prefix", FakeServicePrefix, "--namespace", FakeNamespace, "--range", "0,0", "--load-tool", "wrk", "--load-concurrency", FakeLoadConcurrency, "--load-duration", FakeLoadDuration, "--verbose", "--output", "./")
		assert.NilError(t, err)
	})
}

func Test_getSvcPods(t *testing.T) {
	readyTime := time.Now()
	readyDuration := time.Second * 10
	createTime := readyTime.Add(-readyDuration)

	fakeCtx := context.TODO()
	fakePod1 := getFakePod(FakeServiceName+"-00001", FakeNamespace, map[string]string{"serving.knative.dev/service": FakeServiceName}, createTime, readyTime)
	fakePod2 := getFakePod(FakeServiceName+"-00002", FakeNamespace, map[string]string{"serving.knative.dev/service": FakeServiceName}, createTime, readyTime)
	fakePodList := corev1.PodList{
		Items: []corev1.Pod{
			fakePod1,
			fakePod2,
		},
	}
	client := k8sfake.NewSimpleClientset(&fakePodList)
	var nilPodList []corev1.Pod

	type args struct {
		ctx       context.Context
		params    *pkg.PerfParams
		namespace string
		svcName   string
	}
	tests := []struct {
		name        string
		args        args
		wantPodList []corev1.Pod
		wantErr     bool
	}{
		{
			name: "get pod list",
			args: args{
				ctx: fakeCtx,
				params: &pkg.PerfParams{
					ClientSet: client,
				},
				namespace: FakeNamespace,
				svcName:   FakeServiceName,
			},
			wantPodList: fakePodList.Items,
			wantErr:     false,
		},
		{
			name: "get nil pod list due to namespace error",
			args: args{
				ctx: fakeCtx,
				params: &pkg.PerfParams{
					ClientSet: client,
				},
				namespace: "ktest-000",
				svcName:   FakeServiceName,
			},
			wantPodList: nilPodList,
			wantErr:     false,
		},
		{
			name: "get nil pod list due to svc name error",
			args: args{
				ctx: fakeCtx,
				params: &pkg.PerfParams{
					ClientSet: client,
				},
				namespace: FakeNamespace,
				svcName:   "ktest-1",
			},
			wantPodList: nilPodList,
			wantErr:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPodList, err := getSvcPods(tt.args.ctx, tt.args.params, tt.args.namespace, tt.args.svcName)
			if (err != nil) != tt.wantErr {
				t.Errorf("getSvcPods() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotPodList, tt.wantPodList) {
				t.Errorf("getSvcPods() gotPodList = %v, want %v", gotPodList, tt.wantPodList)
			}
		})
	}
}

func Test_loadCmdBuilder(t *testing.T) {
	type args struct {
		inputs    pkg.LoadArgs
		endpoint  string
		namespace string
		svc       *servingv1.Service
	}

	fakeService := getFakeServingService(FakeServiceName, FakeNamespace)

	inputsHey := pkg.LoadArgs{
		SvcPrefix:       FakeServicePrefix,
		SvcRange:        "0,0",
		Namespace:       FakeNamespace,
		LoadTool:        "hey",
		LoadConcurrency: FakeLoadConcurrency,
		LoadDuration:    FakeLoadDuration,
	}

	cmdHey, _, err := loadCmdBuilder(inputsHey, FakeEndpoint, FakeNamespace, &fakeService)
	if err != nil {
		fmt.Printf("%s\n", err)
		return
	}
	inputsWrk := pkg.LoadArgs{
		SvcPrefix:       FakeServicePrefix,
		SvcRange:        "0,0",
		Namespace:       FakeNamespace,
		LoadTool:        "wrk",
		LoadConcurrency: FakeLoadConcurrency,
		LoadDuration:    FakeLoadDuration,
	}

	cmdWrk, wrkLua, err := loadCmdBuilder(inputsWrk, FakeEndpoint, FakeNamespace, &fakeService)
	if err != nil {
		fmt.Printf("%s\n", err)
		return
	}
	defer func() {
		err := deleteFile(wrkLua)
		if err != nil {
			fmt.Printf("%s\n", err)
		}
	}()

	inputsUnsupportedTool := pkg.LoadArgs{
		SvcPrefix:       FakeServicePrefix,
		SvcRange:        "0,0",
		Namespace:       FakeNamespace,
		LoadTool:        "curl",
		LoadConcurrency: FakeLoadConcurrency,
		LoadDuration:    FakeLoadDuration,
	}

	tests := []struct {
		name    string
		args    args
		want    string
		want1   string
		wantErr bool
	}{
		{
			name: "hey test",
			args: args{
				inputs:    inputsHey,
				endpoint:  FakeEndpoint,
				namespace: FakeNamespace,
				svc:       &fakeService,
			},
			want:    cmdHey,
			want1:   "",
			wantErr: false,
		},
		{
			name: "wrk test",
			args: args{
				inputs:    inputsWrk,
				endpoint:  FakeEndpoint,
				namespace: FakeNamespace,
				svc:       &fakeService,
			},
			want:    cmdWrk,
			want1:   wrkLua,
			wantErr: false,
		},
		{
			name: "unsupported tool",
			args: args{
				inputs:    inputsUnsupportedTool,
				endpoint:  FakeEndpoint,
				namespace: FakeNamespace,
				svc:       &fakeService,
			},
			want:    "",
			want1:   "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := loadCmdBuilder(tt.args.inputs, tt.args.endpoint, tt.args.namespace, tt.args.svc)
			if (err != nil) != tt.wantErr {
				t.Errorf("loadCmdBuilder() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("loadCmdBuilder() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("loadCmdBuilder() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_getReplicasCount(t *testing.T) {
	type args struct {
		loadResult pkg.LoadResult
	}
	tests := []struct {
		name  string
		args  args
		want  int
		want1 []int
	}{
		{
			name: "all good",
			args: args{
				loadResult: pkg.LoadResult{
					Measurment: []pkg.LoadFromZeroResult{
						{
							ServiceName:        FakeServicePrefix + "-0",
							ServiceNamespace:   FakeNamespace,
							TotalReadyReplicas: 2,
							ReplicaResults: []pkg.LoadReplicaResult{
								{
									ReadyReplicasCount: 1,
								},
								{
									ReadyReplicasCount: 2,
								},
							},
						},
						{
							ServiceName:        FakeServicePrefix + "-2",
							ServiceNamespace:   FakeNamespace,
							TotalReadyReplicas: 1,
							ReplicaResults: []pkg.LoadReplicaResult{
								{
									ReadyReplicasCount: 1,
								},
							},
						},
					},
				},
			},
			want:  2,
			want1: []int{2, 1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := getReplicasCount(tt.args.loadResult)
			if got != tt.want {
				t.Errorf("getReplicasCount() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("getReplicasCount() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_getPodResults(t *testing.T) {
	type args struct {
		ctx       context.Context
		params    *pkg.PerfParams
		namespace string
		svc       *servingv1.Service
	}
	readyTime := time.Now()
	readyDuration := time.Second * 10
	createTime := readyTime.Add(-readyDuration)

	fakePod := getFakePod(FakeServiceName+"-00001", FakeNamespace, map[string]string{"serving.knative.dev/service": FakeServiceName}, createTime, readyTime)
	fakePodList := corev1.PodList{
		Items: []corev1.Pod{
			fakePod,
		},
	}

	client := k8sfake.NewSimpleClientset(&fakePodList)
	p := pkg.PerfParams{
		ClientSet: client,
	}
	fakeCtx := context.TODO()
	fakeService := getFakeServingService(FakeServiceName, FakeNamespace)

	fakeArgs := args{
		ctx:       fakeCtx,
		params:    &p,
		namespace: FakeNamespace,
		svc:       &fakeService,
	}

	tests := []struct {
		name    string
		args    args
		want    []pkg.LoadPodResult
		wantErr bool
	}{
		{
			name: "all good",
			args: fakeArgs,
			want: []pkg.LoadPodResult{
				{
					PodCreateTime: metav1.Time{
						Time: createTime,
					}.Rfc3339Copy(),
					PodReadyTime: metav1.Time{
						Time: readyTime,
					}.Rfc3339Copy(),
					PodReadyDuration: readyDuration.Seconds(),
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getPodResults(tt.args.ctx, tt.args.params, tt.args.namespace, tt.args.svc)
			if (err != nil) != tt.wantErr {
				t.Errorf("getPodResults() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getPodResults() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getReplicaResult(t *testing.T) {
	type args struct {
		replicaResults []pkg.LoadReplicaResult
		event          watch.Event
		loadStart      time.Time
	}
	readyTime := time.Now()
	readyDuration := time.Second * 10
	createTime := readyTime.Add(-readyDuration)
	monkey.Patch(time.Now, func() time.Time {
		return readyTime
	})

	fakeDeployment := getFakeDeployment(FakeServiceName+"deployment-00001", FakeNamespace, 1)
	fakeEvent := watch.Event{
		Type:   watch.Modified,
		Object: runtime.Object(&fakeDeployment),
	}

	tests := []struct {
		name string
		args args
		want []pkg.LoadReplicaResult
	}{
		{
			name: "all good",
			args: args{
				replicaResults: []pkg.LoadReplicaResult{},
				event:          fakeEvent,
				loadStart:      createTime,
			},
			want: []pkg.LoadReplicaResult{
				{
					ReadyReplicasCount:   1,
					ReplicaReadyTime:     readyTime,
					ReplicaReadyDuration: readyDuration.Seconds(),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getReplicaResult(tt.args.replicaResults, tt.args.event, tt.args.loadStart)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getReplicaResult() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_deleteFile(t *testing.T) {
	type args struct {
		wrkLua string
	}

	file, err := os.Create("./a.txt")
	if err != nil {
		fmt.Println(err)
	}
	err = file.Close()
	if err != nil {
		fmt.Println(err)
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "file not exist",
			args:    args{wrkLua: "/1y_A_9"},
			wantErr: true,
		},
		{
			name:    "remove successful",
			args:    args{wrkLua: "./a.txt"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := deleteFile(tt.args.wrkLua); (err != nil) != tt.wantErr {
				t.Errorf("deleteWrkLua() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_setLoadFromZeroResult(t *testing.T) {
	type args struct {
		namespace      string
		svc            *servingv1.Service
		replicaResults []pkg.LoadReplicaResult
		podResults     []pkg.LoadPodResult
	}
	readyTime := time.Now()
	readyDuration := time.Second * 10
	createTime := readyTime.Add(-readyDuration)

	fakeService := getFakeServingService(FakeServiceName, FakeNamespace)

	fakeLoadReplicaResult := []pkg.LoadReplicaResult{
		{
			ReadyReplicasCount:   1,
			ReplicaReadyTime:     readyTime,
			ReplicaReadyDuration: readyDuration.Seconds(),
		},
	}
	fakeLoadPodResult := []pkg.LoadPodResult{
		{
			PodCreateTime: metav1.Time{
				Time: createTime,
			}.Rfc3339Copy(),
			PodReadyTime: metav1.Time{
				Time: readyTime,
			}.Rfc3339Copy(),
			PodReadyDuration: readyDuration.Seconds(),
		},
	}

	tests := []struct {
		name string
		args args
		want pkg.LoadFromZeroResult
	}{
		{
			name: "all good",
			args: args{
				namespace:      FakeNamespace,
				svc:            &fakeService,
				replicaResults: fakeLoadReplicaResult,
				podResults:     fakeLoadPodResult,
			},
			want: pkg.LoadFromZeroResult{
				ServiceNamespace:   FakeNamespace,
				ServiceName:        fakeService.Name,
				TotalReadyReplicas: 1,
				TotalReadyPods:     1,
				ReplicaResults:     fakeLoadReplicaResult,
				PodResults:         fakeLoadPodResult,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := setLoadFromZeroResult(tt.args.namespace, tt.args.svc, tt.args.replicaResults, tt.args.podResults); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("setLoadFromZeroResult() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_runLoadFromZero(t *testing.T) {
	// pre-test
	readyTime := time.Now()
	readyDuration := time.Second * 10
	createTime := readyTime.Add(-readyDuration)

	fakeCtx := context.TODO()
	fakeServing := &servingv1fake.FakeServingV1{Fake: &clienttesting.Fake{}}
	servingClient := func() (servingv1client.ServingV1Interface, error) {
		return fakeServing, nil
	}

	fakeService := getFakeServingService(FakeServiceName, FakeNamespace)
	fakeDeployment := getFakeDeployment(FakeServiceName+"-deployment-00001", FakeNamespace, 1)
	fakePod := getFakePod(FakeServiceName+"-00001", FakeNamespace, map[string]string{"serving.knative.dev/service": FakeServiceName}, createTime, readyTime)
	fakePodList := corev1.PodList{
		Items: []corev1.Pod{
			fakePod,
		},
	}

	t.Run("failed to get the cluster endpoint", func(t *testing.T) {
		monkey.Patch(time.Now, func() time.Time {
			return createTime
		})

		fakeIngressSvc, err := getFakeIngressService(FakeIngressServiceName, FakeIngressNamespace, false, "", FakeNodePort)
		if err != nil {
			return
		}
		fakeIngressSvcNilPorts, err := getFakeIngressService(FakeIngressServiceName, FakeIngressNamespace, false, "", "")
		if err != nil {
			return
		}
		fakeIngressSvcHttp, err := getFakeIngressService(FakeIngressServiceName, FakeIngressNamespace, true, "", "")
		if err != nil {
			return
		}

		fakeIngressPod1 := getFakeIngressPod(FakeIngressServiceName+"fbb76f5df-nzw4c", FakeIngressNamespace, map[string]string{"app": FakeIngressServiceName}, "")
		fakeIngressPod2 := getFakeIngressPod(FakeIngressServiceName+"fbb76f5df-nzw4c", FakeIngressNamespace, map[string]string{"app": FakeIngressServiceName}, FakeHostIP)
		client1 := k8sfake.NewSimpleClientset(&fakePodList, &fakeDeployment, &fakeIngressSvc)
		client2 := k8sfake.NewSimpleClientset(&fakePodList, &fakeDeployment, &fakeIngressSvcNilPorts, &fakeIngressPod2)
		client3 := k8sfake.NewSimpleClientset(&fakePodList, &fakeDeployment, &fakeIngressSvcHttp, &fakeIngressPod2)
		client4 := k8sfake.NewSimpleClientset(&fakePodList, &fakeDeployment, &fakeIngressSvcHttp, &fakeIngressPod1)
		p1 := &pkg.PerfParams{
			ClientSet:        client1,
			NewServingClient: servingClient,
		}
		p2 := &pkg.PerfParams{
			ClientSet:        client2,
			NewServingClient: servingClient,
		}
		p3 := &pkg.PerfParams{
			ClientSet:        client3,
			NewServingClient: servingClient,
		}
		p4 := &pkg.PerfParams{
			ClientSet:        client4,
			NewServingClient: servingClient,
		}
		inputs := pkg.LoadArgs{
			SvcRange:        "0,0",
			Namespace:       FakeNamespace,
			SvcPrefix:       FakeServicePrefix,
			Verbose:         true,
			Output:          "/tmp",
			LoadTool:        "hey",
			LoadDuration:    "60s",
			LoadConcurrency: "30",
		}
		_, _, err = runLoadFromZero(fakeCtx, p1, inputs, FakeNamespace, &fakeService)
		assert.ErrorContains(t, err, "ingress pod list is empty")

		_, _, err = runLoadFromZero(fakeCtx, p4, inputs, FakeNamespace, &fakeService)
		assert.ErrorContains(t, err, "host IP of the ingress pod is empty")

		_, _, err = runLoadFromZero(fakeCtx, p2, inputs, FakeNamespace, &fakeService)
		assert.ErrorContains(t, err, "port list of ingress service is empty")

		_, _, err = runLoadFromZero(fakeCtx, p3, inputs, FakeNamespace, &fakeService)
		assert.ErrorContains(t, err, "http2 port of ingress service not found")
	})

	t.Run("load test tool error", func(t *testing.T) {
		monkey.Patch(time.Now, func() time.Time {
			return createTime
		})

		fakeIngressSvc, err := getFakeIngressService(FakeIngressServiceName, FakeIngressNamespace, false, "", FakeNodePort)
		if err != nil {
			return
		}
		fakeIngressPod := getFakeIngressPod(FakeIngressServiceName+"fbb76f5df-nzw4c", FakeIngressNamespace, map[string]string{"app": FakeIngressServiceName}, FakeHostIP)

		client := k8sfake.NewSimpleClientset(&fakePodList, &fakeDeployment, &fakeIngressSvc, &fakeIngressPod)

		p := &pkg.PerfParams{
			ClientSet:        client,
			NewServingClient: servingClient,
		}
		inputs := pkg.LoadArgs{
			SvcRange:        "0,0",
			Namespace:       FakeNamespace,
			SvcPrefix:       FakeServicePrefix,
			Verbose:         true,
			Output:          "/tmp",
			LoadTool:        "curl",
			LoadDuration:    "60s",
			LoadConcurrency: "30",
		}
		_, _, err = runLoadFromZero(fakeCtx, p, inputs, FakeNamespace, &fakeService)
		assert.ErrorContains(t, err, "kperf only support hey and wrk now")
	})
}

func getFakeServingService(name string, ns string) (fakeServingService servingv1.Service) {
	fakeServingService = servingv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
			UID:       "cccccccc-cccc-cccc-cccc-cccccccccccc",
		},
		Status: servingv1.ServiceStatus{
			RouteStatusFields: servingv1.RouteStatusFields{
				URL: &apis.URL{Host: name + "." + ns + ".example.com"},
			},
		},
	}
	return fakeServingService
}

func getFakeDeployment(name string, ns string, readyReplicas int) (fakeDeployment v1.Deployment) {
	fakeDeployment = v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Status: v1.DeploymentStatus{
			ReadyReplicas: int32(readyReplicas),
		},
	}
	return fakeDeployment
}

func getFakePod(name string, namespace string, fakeLabels map[string]string, createTime time.Time, readyTime time.Time) (fakePod corev1.Pod) {
	fakePod = corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    fakeLabels,
			CreationTimestamp: metav1.Time{
				Time: createTime,
			},
		},
		Status: corev1.PodStatus{
			Conditions: []corev1.PodCondition{
				{
					Type:   corev1.PodReady,
					Status: corev1.ConditionTrue,
					LastTransitionTime: metav1.Time{
						Time: readyTime,
					},
				},
			},
		},
	}
	return fakePod
}

func getFakeIngressService(name string, namespace string, errorPorts bool, ip string, nodePort string) (fakeIngressSvc corev1.Service, err error) {
	fakeIngressSvc = corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: corev1.ServiceSpec{
			ExternalIPs: []string{},
			Ports:       []corev1.ServicePort{},
		},
		Status: corev1.ServiceStatus{
			LoadBalancer: corev1.LoadBalancerStatus{
				Ingress: []corev1.LoadBalancerIngress{},
			},
		},
	}
	if nodePort == "" && ip == "" && !errorPorts {
		return fakeIngressSvc, nil
	}
	if nodePort != "" && ip == "" && !errorPorts {
		port, err := strconv.ParseInt(nodePort, 10, 32)
		if err != nil {
			fmt.Println(err)
			return fakeIngressSvc, err
		}
		fakeNodePort := corev1.ServicePort{
			Name:     "http2",
			NodePort: int32(port),
		}
		fakeIngressSvc.Spec.Ports = append(fakeIngressSvc.Spec.Ports, fakeNodePort)
	}
	if nodePort == "" && ip == "" && errorPorts { // set one port but not http2
		port, err := strconv.ParseInt(FakeNodePort, 10, 32)
		if err != nil {
			fmt.Println(err)
			return fakeIngressSvc, err
		}
		fakeNodePort := corev1.ServicePort{
			Name:     "http",
			NodePort: int32(port),
		}
		fakeIngressSvc.Spec.Ports = append(fakeIngressSvc.Spec.Ports, fakeNodePort)
	}

	return fakeIngressSvc, err
}

func getFakeIngressPod(name string, ns string, fakeLabels map[string]string, hostIP string) (fakeIngressPod corev1.Pod) {
	fakeIngressPod = corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
			Labels:    fakeLabels,
		},
	}
	if hostIP != "" {
		fakeIngressPod.Status.HostIP = hostIP
	}
	return fakeIngressPod
}
