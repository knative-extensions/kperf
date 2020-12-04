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
	"sync/atomic"
	"testing"
	"time"

	"gotest.tools/assert"
	"knative.dev/kperf/pkg/generator"
)

func TestBatchCleaner(t *testing.T) {

	var cleanFuncCaled uint64
	cleanFunc := func(ns, name string) {
		atomic.AddUint64(&cleanFuncCaled, 1)
		time.Sleep(1 * time.Second)
	}

	t.Run("empty kn service list", func(t *testing.T) {
		cleanFuncCaled = 0
		start := time.Now().Unix()
		generator.NewBatchCleaner([][2]string{}, 2, cleanFunc).Clean()
		duration := time.Now().Unix() - start
		// should complete immediately
		assert.Assert(t, duration < 1)
		assert.Assert(t, cleanFuncCaled == 0)
	})

	t.Run("nonempty kn service list", func(t *testing.T) {
		cleanFuncCaled = 0
		start := time.Now().Unix()
		generator.NewBatchCleaner([][2]string{{"ns-1", "ksvc-1"}}, 2, cleanFunc).Clean()
		duration := time.Now().Unix() - start
		// should complete in 1 second
		assert.Assert(t, duration >= 1 && duration <= 2)
		assert.Assert(t, cleanFuncCaled == 1)
	})

}
