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

package version

import (
	"fmt"
	"testing"

	"gotest.tools/assert"
	"knative.dev/kperf/pkg/testutil"
)

var versionOutputTemplate = `Version:      %s
Build Date:   %s
Git Revision: %s
`

const (
	fakeVersion     = "fake-version"
	fakeBuildDate   = "fake-build-date"
	fakeGitRevision = "fake-git-revision"
)

func TestVersionOutput(t *testing.T) {
	Version = fakeVersion
	BuildDate = fakeBuildDate
	GitRevision = fakeGitRevision
	expectedOutput := fmt.Sprintf(versionOutputTemplate, fakeVersion, fakeBuildDate, fakeGitRevision)

	versionCmd := NewVersionCommand()
	out, err := testutil.ExecuteCommand(versionCmd)
	assert.NilError(t, err)
	assert.Equal(t, expectedOutput, out)
}
