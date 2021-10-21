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
	"testing"

	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"knative.dev/kperf/pkg"
	"knative.dev/kperf/pkg/testutil"
	servingv1client "knative.dev/serving/pkg/client/clientset/versioned/typed/serving/v1"
	servingv1fake "knative.dev/serving/pkg/client/clientset/versioned/typed/serving/v1/fake"
)

func TestCleanServicesFunc(t *testing.T) {
	tests := []struct {
		name      string
		cleanArgs pkg.CleanArgs
	}{
		{
			name: "should clean services in namespace",
			cleanArgs: pkg.CleanArgs{
				Namespace: "test-kperf-1",
			},
		},
		{
			name: "should clean services in namespace range",
			cleanArgs: pkg.CleanArgs{
				NamespacePrefix: "test-kperf",
				NamespaceRange:  "1,2",
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-kperf-1",
				},
			}
			client := k8sfake.NewSimpleClientset(ns)
			fakeServing := &servingv1fake.FakeServingV1{Fake: &client.Fake}
			servingClient := func() (servingv1client.ServingV1Interface, error) {
				return fakeServing, nil
			}

			p := &pkg.PerfParams{
				ClientSet:        client,
				NewServingClient: servingClient,
			}
			err := CleanServices(p, tc.cleanArgs)
			assert.NilError(t, err)
		})
	}
}

func TestNewServiceCleanCommand(t *testing.T) {
	t.Run("incompleted or wrong args for service clean", func(t *testing.T) {
		client := k8sfake.NewSimpleClientset()
		fakeServing := &servingv1fake.FakeServingV1{Fake: &client.Fake}
		servingClient := func() (servingv1client.ServingV1Interface, error) {
			return fakeServing, nil
		}

		p := &pkg.PerfParams{
			ClientSet:        client,
			NewServingClient: servingClient,
		}
		cmd := NewServiceCleanCommand(p)

		_, err := testutil.ExecuteCommand(cmd)
		assert.ErrorContains(t, err, "both namespace and namespace-prefix are empty")

		_, err = testutil.ExecuteCommand(cmd, "--namespace-prefix", "test-kperf-1", "--namespace-range", "2,1")
		assert.ErrorContains(t, err, "failed to parse namespace range 2,1")

		_, err = testutil.ExecuteCommand(cmd, "--namespace-prefix", "test-kperf", "--namespace-range", "x,y")
		assert.ErrorContains(t, err, "strconv.Atoi: parsing \"x\": invalid syntax")

		_, err = testutil.ExecuteCommand(cmd, "--namespace-prefix", "test-kperf", "--namespace-range", "1,y")
		assert.ErrorContains(t, err, "strconv.Atoi: parsing \"y\": invalid syntax")

		_, err = testutil.ExecuteCommand(cmd, "--namespace-prefix", "test-kperf", "--namespace-range", "1")
		assert.ErrorContains(t, err, "expected range like 1,500, given 1")

		_, err = testutil.ExecuteCommand(cmd, "--namespace-prefix", "test-kperf-1", "--namespace-range", "1,2")
		assert.ErrorContains(t, err, "no namespace found with prefix test-kperf-1")
	})

	t.Run("clean service as expected", func(t *testing.T) {
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-kperf-1",
			},
		}
		client := k8sfake.NewSimpleClientset(ns)
		fakeServing := &servingv1fake.FakeServingV1{Fake: &client.Fake}
		servingClient := func() (servingv1client.ServingV1Interface, error) {
			return fakeServing, nil
		}

		p := &pkg.PerfParams{
			ClientSet:        client,
			NewServingClient: servingClient,
		}

		cmd := NewServiceCleanCommand(p)
		_, err := testutil.ExecuteCommand(cmd, "--namespace", "test-kperf-1")
		assert.NilError(t, err)

		_, err = testutil.ExecuteCommand(cmd, "--namespace-prefix", "test-kperf", "--namespace-range", "1,2")
		assert.NilError(t, err)
	})

	t.Run("failed to clean services", func(t *testing.T) {
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-kperf-2",
			},
		}
		client := k8sfake.NewSimpleClientset(ns)
		fakeServing := &servingv1fake.FakeServingV1{Fake: &client.Fake}
		servingClient := func() (servingv1client.ServingV1Interface, error) {
			return fakeServing, nil
		}

		p := &pkg.PerfParams{
			ClientSet:        client,
			NewServingClient: servingClient,
		}

		cmd := NewServiceCleanCommand(p)
		_, err := testutil.ExecuteCommand(cmd, "--namespace", "test-kperf-1")
		assert.ErrorContains(t, err, "namespaces \"test-kperf-1\" not found")

		cmd = NewServiceCleanCommand(p)
		_, err = testutil.ExecuteCommand(cmd, "--namespace-prefix", "test-kperf-1", "--namespace-range", "1,2")
		assert.ErrorContains(t, err, "no namespace found with prefix test-kperf-1")
	})

	t.Run("clean generated ksvc with namespace flag", func(t *testing.T) {
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-kperf-prefix-1",
			},
		}
		client := k8sfake.NewSimpleClientset(ns)
		fakeServing := &servingv1fake.FakeServingV1{Fake: &client.Fake}
		servingClient := func() (servingv1client.ServingV1Interface, error) {
			return fakeServing, nil
		}

		p := &pkg.PerfParams{
			ClientSet:        client,
			NewServingClient: servingClient,
		}

		cmd := NewServiceCleanCommand(p)
		_, err := testutil.ExecuteCommand(cmd, "--namespace", "test-kperf-prefix-1", "--svc-prefix", "test-ksvc")
		assert.NilError(t, err)
	})
}
