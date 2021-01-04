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

package generator_test

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"gotest.tools/assert"
	"knative.dev/kperf/pkg/generator"
)

func TestBatchGeneratorExit(t *testing.T) {
	count := 8
	failIndex := 3
	generateFunc := func(ns string, index int) (string, string) {
		return ns, fmt.Sprintf("%s-%d", ns, index)
	}
	postGeneratorFunc := func(ns, name string) error {
		if strings.Contains(name, "-"+strconv.Itoa(failIndex)) {
			return errors.New("fake error")
		}
		return nil
	}

	if os.Getenv("RUN_IN_COMMAND") == "true" {
		generator.NewBatchGenerator(time.Duration(1)*time.Second, count, 2, 2, []string{"ns1", "ns2"}, generateFunc, postGeneratorFunc).Generate()
		return
	}
	// Run the test in a subprocess
	cmd := exec.Command(os.Args[0], "-test.run=TestBatchGeneratorExit")
	cmd.Env = append(os.Environ(), "RUN_IN_COMMAND=true")
	start := time.Now().Unix()
	err := cmd.Run()
	duration := time.Now().Unix() - start
	// Cast the error as *exec.ExitError and compare the result
	e, ok := err.(*exec.ExitError)
	expectedErrorString := "exit status 1"
	assert.Equal(t, ok, true)
	assert.ErrorContains(t, e, expectedErrorString)
	// should complete ahead of scheduled 4 (count/batch) seconds
	assert.Assert(t, duration < 4)
}
