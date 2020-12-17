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

set -o pipefail

# Dir where this script is located
basedir() {
  # Default is current directory
  local script=${BASH_SOURCE[0]}

  # Resolve symbolic links
  if [ -L "$script" ]; then
    if readlink -f "$script" >/dev/null 2>&1; then
      script=$(readlink -f "$script")
    elif readlink "$script" >/dev/null 2>&1; then
      script=$(readlink "$script")
    elif realpath "$script" >/dev/null 2>&1; then
      script=$(realpath "$script")
    else
      echo "ERROR: Cannot resolve symbolic link $script"
      exit 1
    fi
  fi

  local dir=$(dirname "$script")
  local full_dir=$(cd "${dir}/.." && pwd)
  echo "${full_dir}"
}

export GO111MODULE=on
export GOPROXY=direct
export GOFLAGS=" -mod=vendor"
export BASE_DIR=$(basedir)

source ${BASE_DIR}/hack/build-funcs.sh

check_license
go_fmt
go_build
go_test
