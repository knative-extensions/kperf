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

package core

import (
	"strings"
	"testing"

	"gotest.tools/v3/assert"

	"knative.dev/kperf/pkg/testutil"
)

func TestNewPerfCommand(t *testing.T) {
	t.Run("check subcommands", func(t *testing.T) {
		expectedSubCommands := []string{
			"help",
			"version",
			"service",
		}

		cmd := NewPerfCommand()
		assert.Check(t, cmd.HasSubCommands())
		assert.Equal(t, len(expectedSubCommands), len(cmd.Commands()))

		for _, e := range expectedSubCommands {
			_, _, err := cmd.Find([]string{e})
			assert.NilError(t, err, "root command should have subcommand %q", e)
		}
	})

	t.Run("make sure usage has kperf", func(t *testing.T) {
		cmd := NewPerfCommand()
		output, err := testutil.ExecuteCommand(cmd)
		assert.NilError(t, err)
		assert.Check(t, strings.Contains(output, "Usage:\n  kperf [command]\n"), "invalid usage %q", output)
	})

	t.Run("check kperf help", func(t *testing.T) {
		cmd := NewPerfCommand()
		_, err := testutil.ExecuteCommand(cmd, "help")
		assert.NilError(t, err)
	})

	t.Run("run unknown command", func(t *testing.T) {
		cmd := NewPerfCommand()
		_, err := testutil.ExecuteCommand(cmd, "test-command")
		assert.ErrorContains(t, err, "unknown command \"test-command\" for \"kperf\"")
	})
}
