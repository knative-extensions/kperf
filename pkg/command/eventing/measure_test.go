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

package eventing

import (
	"testing"
)

func TestNewEventingMeasureCommand(t *testing.T) {
	// t.Run("incompleted or wrong args for service measure", func(t *testing.T) {
	// 	client := k8sfake.NewSimpleClientset()

	// 	p := &pkg.PerfParams{
	// 		ClientSet: client,
	// 	}

	// 	cmd := NewEventingMeasureCommand(p)

	// 	_, err := testutil.ExecuteCommand(cmd)
	// 	assert.ErrorContains(t, err, "'eventing measure' requires flag(s)")

	// 	_, err = testutil.ExecuteCommand(cmd, "--range", "1200", "--namespace", "ns")
	// 	assert.ErrorContains(t, err, "expected range like 1,500, given 1200")

	// 	_, err = testutil.ExecuteCommand(cmd, "--range", "1200", "--namespace-prefix", "ns", "--namespace-range", "1,2")
	// 	assert.ErrorContains(t, err, "expected range like 1,500, given 1200")

	// 	_, err = testutil.ExecuteCommand(cmd, "--range", "x,y", "--namespace", "ns")
	// 	assert.ErrorContains(t, err, "strconv.Atoi: parsing \"x\": invalid syntax")

	// 	_, err = testutil.ExecuteCommand(cmd, "--range", "x,y", "--namespace-prefix", "ns", "--namespace-range", "1,2")
	// 	assert.ErrorContains(t, err, "strconv.Atoi: parsing \"x\": invalid syntax")

	// 	_, err = testutil.ExecuteCommand(cmd, "--range", "1,y", "--namespace", "ns")
	// 	assert.ErrorContains(t, err, "strconv.Atoi: parsing \"y\": invalid syntax")

	// 	_, err = testutil.ExecuteCommand(cmd, "--range", "1,y", "--namespace-prefix", "ns", "--namespace-range", "1,2")
	// 	assert.ErrorContains(t, err, "strconv.Atoi: parsing \"y\": invalid syntax")
	//})

}
