#!/bin/bash

# Copyright 2021 The Knative Authors
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

set -o pipefail

TOOLS_DIR=hack/tools
TOOLS_BIN_DIR=hack/tools/bin
WOKE_VERSION=v0.13.0

mkdir -p ${TOOLS_BIN_DIR}

echo '🐶 Installing reviewdog ... https://github.com/reviewdog/reviewdog'
curl -sfL https://raw.githubusercontent.com/reviewdog/reviewdog/master/install.sh | sh -s -- -b "${TOOLS_BIN_DIR}" 2>&1

echo 'Installing misspell ... https://github.com/client9/misspell'
curl -sfL https://raw.githubusercontent.com/client9/misspell/master/install-misspell.sh | sh -s -- -b "${TOOLS_BIN_DIR}" 2>&1

echo 'Installing woke ... https://github.com/get-woke/woke'
curl -sfL https://raw.githubusercontent.com/get-woke/woke/main/install.sh | sh -s -- -b "${TOOLS_BIN_DIR}" "${WOKE_VERSION}" 2>&1

echo 'Installing golangci-lint ...'
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b "${TOOLS_BIN_DIR}" v1.43.0 2>&1
