#!/bin/bash

# Copyright 2020 The Knative Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Calculate the base directory
function basedir() {
  # Default is current directory
  local script=${BASH_SOURCE[0]}
  local dir=$(dirname "$script")
  local full_dir=$(cd "${dir}/.." && pwd)
  echo ${full_dir}
}

# Environment variable which can be used my plugins
export BASE_DIR=$(basedir)
export TEST_INFRA_SCRIPTS="$BASE_DIR/test/test-infra"
export KPERF_HACK_SCRIPTS="$BASE_DIR/hack"
