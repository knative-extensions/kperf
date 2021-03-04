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
	"context"
	"testing"

	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"
	"knative.dev/kperf/pkg"
	"knative.dev/kperf/pkg/testutil"
	networkingv1alpha1 "knative.dev/networking/pkg/client/clientset/versioned/typed/networking/v1alpha1"
	fakenetworkingv1alpha1 "knative.dev/networking/pkg/client/clientset/versioned/typed/networking/v1alpha1/fake"
	autoscalingv1client "knative.dev/serving/pkg/client/clientset/versioned/typed/autoscaling/v1alpha1"
	autoscalingv1fake "knative.dev/serving/pkg/client/clientset/versioned/typed/autoscaling/v1alpha1/fake"

	servingv1client "knative.dev/serving/pkg/client/clientset/versioned/typed/serving/v1"
	servingv1fake "knative.dev/serving/pkg/client/clientset/versioned/typed/serving/v1/fake"
)

func TestNewServiceMeasureCommand(t *testing.T) {
	t.Run("incompleted or wrong args for service measure", func(t *testing.T) {
		client := k8sfake.NewSimpleClientset()

		p := &pkg.PerfParams{
			ClientSet: client,
		}

		cmd := NewServiceMeasureCommand(p)

		_, err := testutil.ExecuteCommand(cmd)
		assert.ErrorContains(t, err, "'service measure' requires flag(s)")

		_, err = testutil.ExecuteCommand(cmd, "--range", "1200", "--namespace", "ns")
		assert.ErrorContains(t, err, "expected range like 1,500, given 1200")

		_, err = testutil.ExecuteCommand(cmd, "--range", "1200", "--namespace-prefix", "ns", "--namespace-range", "1,2")
		assert.ErrorContains(t, err, "expected range like 1,500, given 1200")

		_, err = testutil.ExecuteCommand(cmd, "--range", "x,y", "--namespace", "ns")
		assert.ErrorContains(t, err, "strconv.Atoi: parsing \"x\": invalid syntax")

		_, err = testutil.ExecuteCommand(cmd, "--range", "x,y", "--namespace-prefix", "ns", "--namespace-range", "1,2")
		assert.ErrorContains(t, err, "strconv.Atoi: parsing \"x\": invalid syntax")

		_, err = testutil.ExecuteCommand(cmd, "--range", "1,y", "--namespace", "ns")
		assert.ErrorContains(t, err, "strconv.Atoi: parsing \"y\": invalid syntax")

		_, err = testutil.ExecuteCommand(cmd, "--range", "1,y", "--namespace-prefix", "ns", "--namespace-range", "1,2")
		assert.ErrorContains(t, err, "strconv.Atoi: parsing \"y\": invalid syntax")
	})

	t.Run("measure service as expected with namespace flag", func(t *testing.T) {
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns1",
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

		cmd := NewServiceMeasureCommand(p)
		_, err := testutil.ExecuteCommand(cmd, "--svc-prefix", "svc", "--namespace", "ns1", "--range", "1,1")
		assert.NilError(t, err)
	})

	t.Run("measure service as expected with namespace prefix flag", func(t *testing.T) {
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns-1",
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

		cmd := NewServiceMeasureCommand(p)
		_, err := testutil.ExecuteCommand(cmd, "--svc-prefix", "svc", "--namespace-prefix", "ns", "--namespace-range", "1,1")
		assert.ErrorContains(t, err, "no service found to measure")
	})
}

func TestSortSlice(t *testing.T) {
	rows := [][]string{{"test-2"}, {"test-1"}}
	sortSlice(rows)
	assert.DeepEqual(t, [][]string{{"test-1"}, {"test-2"}}, rows)
}

func TestGetPodCondition(t *testing.T) {
	t.Run("get pod condition when pod is scheduled", func(t *testing.T) {
		podCondition := &corev1.PodCondition{
			Type: corev1.PodScheduled,
		}
		podStatus := &corev1.PodStatus{
			Conditions: []corev1.PodCondition{
				*podCondition,
			},
		}
		i, condition := getPodCondition(podStatus, corev1.PodScheduled)
		assert.Equal(t, 0, i)
		assert.Equal(t, corev1.PodScheduled, condition.Type)
	})

	t.Run("get pod condition when pod isn't scheduled", func(t *testing.T) {
		podCondition := &corev1.PodCondition{
			Type: corev1.PodInitialized,
		}
		podStatus := &corev1.PodStatus{
			Conditions: []corev1.PodCondition{
				*podCondition,
			},
		}
		i, condition := getPodCondition(podStatus, corev1.PodScheduled)
		assert.Equal(t, -1, i)
		assert.Equal(t, (*corev1.PodCondition)(nil), condition)
	})

	t.Run("get pod condition when pod status is nil", func(t *testing.T) {
		podCondition := (*corev1.PodCondition)(nil)
		i, condition := getPodCondition(nil, corev1.PodScheduled)
		assert.Equal(t, -1, i)
		assert.Equal(t, podCondition, condition)
	})
}

func TestGetContainerStatus(t *testing.T) {
	t.Run("get container status seccussfully", func(t *testing.T) {
		var containerStatus []corev1.ContainerStatus
		container := corev1.ContainerStatus{
			Name: "user-container",
		}
		containerStatus = append(containerStatus, container)
		s, status := getContainerStatus(containerStatus, "user-container")
		assert.Equal(t, container.Name, s.Name)
		assert.Equal(t, true, status)
	})

	t.Run("get container status when the condition is not present", func(t *testing.T) {
		var containerStatus []corev1.ContainerStatus
		s, status := getContainerStatus(containerStatus, "queue-proxy")
		assert.Equal(t, (*corev1.ContainerStatus)(nil), s)
		assert.Equal(t, false, status)
	})
}

func TestGetKnativeVersion(t *testing.T) {
	t.Run("get knative serving and eventing version", func(t *testing.T) {
		servingNs := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "knative-serving",
				Labels: map[string]string{"serving.knative.dev/release": "v0.20.0"},
			},
		}
		eventingNs := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "knative-eventing",
				Labels: map[string]string{"eventing.knative.dev/release": "v0.20.0"},
			},
		}
		client := k8sfake.NewSimpleClientset(servingNs, eventingNs)

		p := &pkg.PerfParams{
			ClientSet: client,
		}
		version := getKnativeVersion(p)
		assert.Equal(t, "0.20.0", version["serving"])
		assert.Equal(t, "0.20.0", version["eventing"])
	})

	t.Run("failed to get knative serving and eventing version", func(t *testing.T) {
		client := k8sfake.NewSimpleClientset()
		fakeServing := &servingv1fake.FakeServingV1{Fake: &client.Fake}
		servingClient := func() (servingv1client.ServingV1Interface, error) {
			return fakeServing, nil
		}

		p := &pkg.PerfParams{
			ClientSet:        client,
			NewServingClient: servingClient,
		}
		version := getKnativeVersion(p)
		assert.Equal(t, "Unknown", version["serving"])
		assert.Equal(t, "Unknown", version["eventing"])
	})
}

func TestGetIngressController(t *testing.T) {
	t.Run("get knative ingress controller with version", func(t *testing.T) {
		servingNs := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "knative-serving",
			},
		}

		istioNs := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "istio-system",
			},
		}

		client := k8sfake.NewSimpleClientset(servingNs, istioNs)
		p := &pkg.PerfParams{
			ClientSet: client,
		}

		configMapKnative := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: "config-network",
			},
			Data: map[string]string{"ingress.class": "istio.ingress.networking.knative.dev"},
		}
		p.ClientSet.CoreV1().ConfigMaps("knative-serving").Create(context.TODO(), configMapKnative, metav1.CreateOptions{})

		configMapIstio := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "istio",
				Labels: map[string]string{"operator.istio.io/version": "1.7.3"},
			},
			Data: map[string]string{"ingress.class": "istio.ingress.networking.knative.dev"},
		}
		p.ClientSet.CoreV1().ConfigMaps("istio-system").Create(context.TODO(), configMapIstio, metav1.CreateOptions{})

		ingressController := getIngressController(p)
		assert.Equal(t, "Istio", ingressController["ingressController"])
		assert.Equal(t, "1.7.3", ingressController["version"])
	})

	t.Run("get knative ingress controller without version", func(t *testing.T) {
		servingNs := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "knative-serving",
			},
		}

		client := k8sfake.NewSimpleClientset(servingNs)
		p := &pkg.PerfParams{
			ClientSet: client,
		}
		configMapKnative := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: "config-network",
			},
			Data: map[string]string{"ingress.class": "istio.ingress.networking.knative.dev"},
		}
		p.ClientSet.CoreV1().ConfigMaps("knative-serving").Create(context.TODO(), configMapKnative, metav1.CreateOptions{})

		ingressController := getIngressController(p)
		assert.Equal(t, "Istio", ingressController["ingressController"])
		assert.Equal(t, "Unknown", ingressController["version"])
	})

	t.Run("get unknown knative ingress controller", func(t *testing.T) {
		servingNs := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "knative-serving",
			},
		}

		client := k8sfake.NewSimpleClientset(servingNs)
		p := &pkg.PerfParams{
			ClientSet: client,
		}
		configMapKnative := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: "config-network",
			},
		}
		p.ClientSet.CoreV1().ConfigMaps("knative-serving").Create(context.TODO(), configMapKnative, metav1.CreateOptions{})

		ingressController := getIngressController(p)
		assert.Equal(t, "Unknown", ingressController["ingressController"])
		assert.Equal(t, "Unknown", ingressController["version"])
	})
}
