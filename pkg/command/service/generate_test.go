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
	"os"
	"path/filepath"
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

func TestNewServiceGenerateCommand(t *testing.T) {
	t.Run("incompleted or wrong args for service generate", func(t *testing.T) {
		client := k8sfake.NewSimpleClientset()
		fakeServing := &servingv1fake.FakeServingV1{Fake: &client.Fake}
		servingClient := func() (servingv1client.ServingV1Interface, error) {
			return fakeServing, nil
		}

		p := &pkg.PerfParams{
			ClientSet:        client,
			NewServingClient: servingClient,
		}
		cmd := NewServiceGenerateCommand(p)
		_, err := testutil.ExecuteCommand(cmd)
		assert.ErrorContains(t, err, "required flag(s)")
		assert.ErrorContains(t, err, "batch")
		assert.ErrorContains(t, err, "interval")
		assert.ErrorContains(t, err, "number")

		_, err = testutil.ExecuteCommand(cmd, "-b", "1")
		assert.ErrorContains(t, err, "required flag(s)")
		assert.ErrorContains(t, err, "interval")
		assert.ErrorContains(t, err, "number")

		_, err = testutil.ExecuteCommand(cmd, "-b", "1", "-i", "1")
		assert.ErrorContains(t, err, "required flag(s) \"number\"")

		_, err = testutil.ExecuteCommand(cmd, "-b", "1", "-i", "1", "--min-scale", "1")
		assert.ErrorContains(t, err, "required flag(s) \"number\"")

		_, err = testutil.ExecuteCommand(cmd, "-b", "1", "-i", "1", "--min-scale", "1", "--max-scale", "2")
		assert.ErrorContains(t, err, "required flag(s) \"number\"")

		_, err = testutil.ExecuteCommand(cmd, "-b", "1", "-i", "1", "--min-scale", "1", "--max-scale", "2", "--number", "1")
		assert.ErrorContains(t, err, "namespace default not found, please create one")

		_, err = testutil.ExecuteCommand(cmd, "--namespace-prefix", "test-kperf", "--namespace-range", "2,1")
		assert.ErrorContains(t, err, "failed to parse namespace range")

		_, err = testutil.ExecuteCommand(cmd, "--namespace-prefix", "test-kperf", "--namespace-range", "x,y")
		assert.ErrorContains(t, err, "strconv.Atoi: parsing \"x\": invalid syntax")

		_, err = testutil.ExecuteCommand(cmd, "--namespace-prefix", "test-kperf", "--namespace-range", "1,y")
		assert.ErrorContains(t, err, "strconv.Atoi: parsing \"y\": invalid syntax")

		_, err = testutil.ExecuteCommand(cmd, "--namespace-prefix", "test-kperf", "--namespace-range", "1")
		assert.ErrorContains(t, err, "expected range like 1,500, given 1")

		_, err = testutil.ExecuteCommand(cmd, "--namespace-prefix", "test-kperf", "--namespace", "test-kperf")
		assert.ErrorContains(t, err, "expected either namespace with prefix & range or only namespace name")
	})

	t.Run("generate service as expected with namespace flag", func(t *testing.T) {
		ns1 := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-kperf-1",
			},
		}
		client := k8sfake.NewSimpleClientset(ns1)
		fakeServing := &servingv1fake.FakeServingV1{Fake: &client.Fake}
		servingClient := func() (servingv1client.ServingV1Interface, error) {
			return fakeServing, nil
		}

		p := &pkg.PerfParams{
			ClientSet:        client,
			NewServingClient: servingClient,
		}

		cmd := NewServiceGenerateCommand(p)
		_, err := testutil.ExecuteCommand(cmd, "-n", "1", "-b", "10", "-i", "10", "--min-scale", "1", "--max-scale", "2", "--namespace", "test-kperf-1")
		assert.NilError(t, err)

		ksvcClient, err := p.NewServingClient()
		assert.NilError(t, err)
		svc, err := ksvcClient.Services("test-kperf-1").Get(context.TODO(), "ksvc-0", metav1.GetOptions{})
		assert.NilError(t, err)
		assert.Equal(t, "ksvc-0", svc.Name)
		targetAnnotations := make(map[string]string)
		targetAnnotations["autoscaling.knative.dev/maxScale"] = "2"
		targetAnnotations["autoscaling.knative.dev/minScale"] = "1"
		resultAnnotations := svc.Spec.ConfigurationSpec.GetTemplate().GetAnnotations()
		assert.DeepEqual(t, targetAnnotations, resultAnnotations)
	})

	t.Run("generate service as expected with namespace prefix flag", func(t *testing.T) {
		ns1 := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-kperf-prefix-1",
			},
		}
		ns2 := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-kperf-prefix-2",
			},
		}
		client := k8sfake.NewSimpleClientset(ns1, ns2)
		fakeServing := &servingv1fake.FakeServingV1{Fake: &client.Fake}
		servingClient := func() (servingv1client.ServingV1Interface, error) {
			return fakeServing, nil
		}

		p := &pkg.PerfParams{
			ClientSet:        client,
			NewServingClient: servingClient,
		}

		cmd := NewServiceGenerateCommand(p)
		_, err := testutil.ExecuteCommand(cmd, "-n", "2", "-b", "10", "-i", "10", "--min-scale", "1", "--max-scale", "2", "--namespace-prefix", "test-kperf-prefix", "--namespace-range", "1,2")
		assert.NilError(t, err)

		ksvcClient, _ := p.NewServingClient()
		svc, _ := ksvcClient.Services("test-kperf-prefix-1").Get(context.TODO(), "ksvc-0", metav1.GetOptions{})
		assert.Equal(t, "ksvc-0", svc.Name)
		targetAnnotations := make(map[string]string)
		targetAnnotations["autoscaling.knative.dev/maxScale"] = "2"
		targetAnnotations["autoscaling.knative.dev/minScale"] = "1"
		resultAnnotations := svc.Spec.ConfigurationSpec.GetTemplate().GetAnnotations()
		assert.DeepEqual(t, targetAnnotations, resultAnnotations)

		svc, _ = ksvcClient.Services("test-kperf-prefix-2").Get(context.TODO(), "ksvc-1", metav1.GetOptions{})
		assert.Equal(t, "ksvc-1", svc.Name)
		targetAnnotations = make(map[string]string)
		targetAnnotations["autoscaling.knative.dev/maxScale"] = "2"
		targetAnnotations["autoscaling.knative.dev/minScale"] = "1"
		resultAnnotations = svc.Spec.ConfigurationSpec.GetTemplate().GetAnnotations()
		assert.DeepEqual(t, targetAnnotations, resultAnnotations)
	})

	t.Run("failed to generate service", func(t *testing.T) {
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

		cmd := NewServiceGenerateCommand(p)
		_, err := testutil.ExecuteCommand(cmd, "-n", "1", "-b", "10", "-i", "10", "--min-scale", "1", "--max-scale", "2", "--namespace", "test-kperf-1")
		assert.ErrorContains(t, err, "namespace test-kperf-1 not found, please create one")

		cmd = NewServiceGenerateCommand(p)
		_, err = testutil.ExecuteCommand(cmd, "-n", "1", "-b", "10", "-i", "10", "--min-scale", "1", "--max-scale", "2", "--namespace-prefix", "test-kperf", "--namespace-range", "1,2")
		assert.ErrorContains(t, err, "namespace test-kperf-1 not found, please create one")
	})

	t.Run("create service from template", func(t *testing.T) {

		ns1 := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-kperf-1",
			},
		}
		client := k8sfake.NewSimpleClientset(ns1)
		fakeServing := &servingv1fake.FakeServingV1{Fake: &client.Fake}
		servingClient := func() (servingv1client.ServingV1Interface, error) {
			return fakeServing, nil
		}

		p := &pkg.PerfParams{
			ClientSet:        client,
			NewServingClient: servingClient,
		}

		templatePath := filepath.Join(t.TempDir(), "template.yaml")
		templateYaml := `apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: hello
  namespace: default
spec:
  template:
    metadata:
      annotations:
        autoscaling.knative.dev/minScale: "5"
        autoscaling.knative.dev/maxScale: "6"
    spec:
      containers:
      - image: gcr.io/knative-samples/helloworld-rust
        env:
          - name: TARGET
            value: "Kperf Test"`

		if err := os.WriteFile(templatePath, []byte(templateYaml), 0700); err != nil {
			t.Fatal(err)
		}

		cmd := NewServiceGenerateCommand(p)
		_, err := testutil.ExecuteCommand(cmd, "-n", "1", "-b", "10", "-i", "10", "--namespace", "test-kperf-1", "--template", templatePath)
		assert.NilError(t, err)

		ksvcClient, err := p.NewServingClient()
		assert.NilError(t, err)
		svc, err := ksvcClient.Services("test-kperf-1").Get(context.TODO(), "ksvc-0", metav1.GetOptions{})
		assert.NilError(t, err)
		assert.Equal(t, "ksvc-0", svc.Name)
		assert.Equal(t, "gcr.io/knative-samples/helloworld-rust", svc.Spec.Template.Spec.Containers[0].Image)
		targetAnnotations := make(map[string]string)
		targetAnnotations["autoscaling.knative.dev/maxScale"] = "6"
		targetAnnotations["autoscaling.knative.dev/minScale"] = "5"
		resultAnnotations := svc.Spec.ConfigurationSpec.GetTemplate().GetAnnotations()
		assert.DeepEqual(t, targetAnnotations, resultAnnotations)

	})
}
