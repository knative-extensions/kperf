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
	"testing"

	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"
	"knative.dev/kperf/pkg"
	"knative.dev/kperf/pkg/testutil"
	networkingv1alpha1 "knative.dev/networking/pkg/client/clientset/versioned/typed/networking/v1alpha1"
	fakenetworkingv1alpha1 "knative.dev/networking/pkg/client/clientset/versioned/typed/networking/v1alpha1/fake"
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
		fakeServing := &servingv1fake.FakeServingV1{Fake: &clienttesting.Fake{}}
		servingClient := func() (servingv1client.ServingV1Interface, error) {
			return fakeServing, nil
		}

		fakeNetworking := &fakenetworkingv1alpha1.FakeNetworkingV1alpha1{Fake: &clienttesting.Fake{}}
		networkingClient := func() (networkingv1alpha1.NetworkingV1alpha1Interface, error) {
			return fakeNetworking, nil
		}

		p := &pkg.PerfParams{
			ClientSet:           client,
			NewServingClient:    servingClient,
			NewNetworkingClient: networkingClient,
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
		fakeServing := &servingv1fake.FakeServingV1{Fake: &clienttesting.Fake{}}
		servingClient := func() (servingv1client.ServingV1Interface, error) {
			return fakeServing, nil
		}

		fakeNetworking := &fakenetworkingv1alpha1.FakeNetworkingV1alpha1{Fake: &clienttesting.Fake{}}
		networkingClient := func() (networkingv1alpha1.NetworkingV1alpha1Interface, error) {
			return fakeNetworking, nil
		}

		p := &pkg.PerfParams{
			ClientSet:           client,
			NewServingClient:    servingClient,
			NewNetworkingClient: networkingClient,
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
