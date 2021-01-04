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
// limitations under the License

package service

import (
	"testing"

	"gotest.tools/assert"
)

func TestNewServiceCmd(t *testing.T) {
	cmd := NewServiceCmd(nil)
	assert.Check(t, cmd.HasSubCommands(), "cmd service should have subcommands")

	_, _, err := cmd.Find([]string{"generate"})
	assert.NilError(t, err, "service command should have generate subcommand")

	_, _, err = cmd.Find([]string{"measure"})
	assert.NilError(t, err, "service command should have measure subcommand")

	_, _, err = cmd.Find([]string{"clean"})
	assert.NilError(t, err, "service command should have clean subcommand")
}
