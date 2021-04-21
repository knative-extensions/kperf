/*
Copyright 2021 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package receiver

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"knative.dev/kperf/pkg"
)

type deploymentOptions struct {
	Env []corev1.EnvVar
}

func installDeployment(p *pkg.PerfParams, name string, namespace string, options deploymentOptions) error {
	d := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: appsv1.SchemeGroupVersion.String(),
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{

				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:    "kperf-eventing-receiver",
							Image:   "TODO", // TODO: what's the story?
							Command: []string{"/kperf"},
							Args:    []string{"eventing", "receiver"},
							Env:     options.Env,
							Ports: []corev1.ContainerPort{{
								Name:          "http",
								ContainerPort: 8081,
							}},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									"cpu":    resource.MustParse("250m"),
									"memory": resource.MustParse("512Mi"),
								},
								Limits: corev1.ResourceList{
									"cpu":    resource.MustParse("2000m"),
									"memory": resource.MustParse("1024Mi"),
								},
							},
						},
					},
				},
			},
		},
	}

	_, err := p.ClientSet.AppsV1().Deployments(namespace).Create(context.Background(), d, metav1.CreateOptions{})
	return err
}
