#!/usr/bin/env bash

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

source "$(dirname $0)"/common.sh
source "$TEST_INFRA_SCRIPTS/presubmit-tests.sh"
source "$KPERF_HACK_SCRIPTS/build-funcs.sh"

function build_tests() {
    go_build
}

function unit_tests() {
    go_test
}

function integration_tests() {
    echo "No integration tests right now"
}

# We use the default integration test runner.
main "$@"
