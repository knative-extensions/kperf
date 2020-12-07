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

package generator_test

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"gotest.tools/assert"
	"knative.dev/kperf/pkg/generator"
)

func TestBatchGenerator(t *testing.T) {
	var generateFuncCaled uint64
	var postGeneratorFuncCalled uint64

	generateFunc := func(ns string, index int) (string, string) {
		atomic.AddUint64(&generateFuncCaled, 1)
		return ns, fmt.Sprintf("%s-%d", ns, index)
	}
	postGeneratorFunc := func(ns, name string) error {
		atomic.AddUint64(&postGeneratorFuncCalled, 1)
		return nil
	}

	t.Run("should complete immediately since 0 ksvc is required to be created", func(t *testing.T) {
		generateFuncCaled = 0
		postGeneratorFuncCalled = 0
		start := time.Now().Unix()
		generator.NewBatchGenerator(time.Duration(1)*time.Second, 0, 2, 2, []string{"ns1", "ns2"}, generateFunc, postGeneratorFunc).Generate()
		duration := time.Now().Unix() - start
		// should complete immediately
		assert.Assert(t, duration < 1)
		assert.Assert(t, generateFuncCaled == 0)
		assert.Assert(t, postGeneratorFuncCalled == 0)
	})

	t.Run("should complete in 4s", func(t *testing.T) {
		generateFuncCaled = 0
		postGeneratorFuncCalled = 0
		start := time.Now().Unix()
		generator.NewBatchGenerator(time.Duration(1)*time.Second, 8, 2, 2, []string{"ns1", "ns2"}, generateFunc, postGeneratorFunc).Generate()
		duration := time.Now().Unix() - start
		// should complete in count/batch = 4 seconds
		assert.Assert(t, duration >= 4 && duration <= 5)
		assert.Assert(t, generateFuncCaled == 8)
		assert.Assert(t, postGeneratorFuncCalled == 8)
	})

	t.Run("should complete in 2s", func(t *testing.T) {
		generateFuncCaled = 0
		postGeneratorFuncCalled = 0
		start := time.Now().Unix()
		generator.NewBatchGenerator(time.Duration(1)*time.Second, 8, 4, 2, []string{"ns1", "ns2"}, generateFunc, postGeneratorFunc).Generate()
		duration := time.Now().Unix() - start
		// should complete in count/batch = 2 seconds
		assert.Assert(t, duration >= 2 && duration <= 3)
		assert.Assert(t, generateFuncCaled == 8)
		assert.Assert(t, postGeneratorFuncCalled == 8)
	})

	t.Run("should complete in 1s", func(t *testing.T) {
		generateFuncCaled = 0
		postGeneratorFuncCalled = 0
		start := time.Now().Unix()
		generator.NewBatchGenerator(time.Duration(1)*time.Second, 8, 8, 16, []string{"ns1", "ns2"}, generateFunc, postGeneratorFunc).Generate()
		duration := time.Now().Unix() - start
		// should complete in count/batch = 1 second
		assert.Assert(t, duration >= 1 && duration <= 2)
		assert.Assert(t, generateFuncCaled == 8)
		assert.Assert(t, postGeneratorFuncCalled == 8)
	})
}
