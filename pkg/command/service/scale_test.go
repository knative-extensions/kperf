// Copyright 2021 The Knative Authors
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
	"testing"

	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"
	"knative.dev/kperf/pkg"
	networkingv1alpha1 "knative.dev/networking/pkg/client/clientset/versioned/typed/networking/v1alpha1"
	fakenetworkingv1alpha1 "knative.dev/networking/pkg/client/clientset/versioned/typed/networking/v1alpha1/fake"
	autoscalingv1client "knative.dev/serving/pkg/client/clientset/versioned/typed/autoscaling/v1alpha1"
	autoscalingv1fake "knative.dev/serving/pkg/client/clientset/versioned/typed/autoscaling/v1alpha1/fake"

	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	servingv1client "knative.dev/serving/pkg/client/clientset/versioned/typed/serving/v1"
	servingv1fake "knative.dev/serving/pkg/client/clientset/versioned/typed/serving/v1/fake"
)

func TestScaleServces(t *testing.T) {
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

	//"--svc-prefix", "svc", "--namespace", "ns1", "--range", "1,1")
	scaleArgs := pkg.ScaleArgs{
		SvcPrefix: "ksvc-",
		Namespace: "ns-1",
		SvcRange:  "1,1",
	}

	getFakeServices := func(servingv1client.ServingV1Interface, []string, string) []ServicesToScale {
		objs := []ServicesToScale{}
		svc := ServicesToScale{
			Service: &servingv1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ksvc-1",
					Namespace: "ns-1",
					UID:       "cccccccc-cccc-cccc-cccc-cccccccccccc",
					Annotations: map[string]string{
						"autoscaling.knative.dev/minScale": "1",
						"autoscaling.knative.dev/maxScale": "2",
					},
				},
			},
			Namespace: "ns-1",
		}

		objs = append(objs, svc)
		return objs
	}

	_, err := scaleAndMeasure(p, scaleArgs, []string{"ns-1"}, getFakeServices)
	assert.NilError(t, err)
}
