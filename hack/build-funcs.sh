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

export SOURCE_DIRS="cmd pkg core"

function build_flags() {
  local now="$(date -u '+%Y-%m-%d %H:%M:%S')"
  local rev="$(git rev-parse --short HEAD)"
  local pkg="knative.dev/kperf/pkg/command/version"
  local version="${TAG:-}"
  if [[ -z "${version}" ]]; then
    # Get the commit, excluding any tags but keeping the "dirty" flag
    local commit="$(git describe --always --dirty --match '^$')"
    [[ -n "${commit}" ]] || abort "error getting the current commit"
    version="v$(date +%Y%m%d)-${commit}"
  fi

  echo "-X '${pkg}.BuildDate=${now}' -X ${pkg}.Version=${version} -X ${pkg}.GitRevision=${rev}"
}

function go_fmt() {
  echo "🧹 ${S}Format"
  find $(echo "${SOURCE_DIRS}") -name "*.go" -print0 | xargs -0 gofmt -s -w
}

function go_pre_build() {
  export PATH=$PATH:$GOPATH/bin
  go mod vendor
  go get -u github.com/jteeuwen/go-bindata/...
  pushd $BASE_DIR > /dev/null
  go-bindata -nometadata -pkg utils -o ./pkg/command/utils/htmltemplatebindata.go ./templates/...
  popd > /dev/null
}

function go_build() {
  echo "🚧 Compile"
  go_pre_build
  go build -mod=mod -ldflags "$(build_flags)" -o kperf ./cmd/...
}

function go_test() {
  local red=""
  local reset=""
  # Use color only when a terminal is set
  if [ -t 1 ]; then
    red="[31m"
    reset="[39m"
  fi

  echo "🧪 ${X}Test"
  go_pre_build
  if ! go test -v ./...; then
    echo "🔥 ${red}Failure${reset}"
    exit 1
  fi
}

function check_license() {
  echo "⚖️ ${S}License"
  local required_keywords=("Authors" "Apache License" "LICENSE-2.0")
  local extensions_to_check=("sh" "go" "yaml" "yml" "json")

  local check_output=$(mktemp /tmp/${PLUGIN}-licence-check.XXXXXX)
  for ext in "${extensions_to_check[@]}"; do
    find . -name "*.$ext" -a \! -path "./vendor/*" -a \! -path "./.*" -a \! -path "./pkg/command/utils/htmltemplatebindata.go" -print0 |
      while IFS= read -r -d '' path; do
        for rword in "${required_keywords[@]}"; do
          if ! grep -q "$rword" "$path"; then
            echo "   $path" >> $check_output
          fi
        done
      done
  done
  if [ -s $check_output ]; then
    echo "🔥 No license header found in:"
    cat $check_output | sort | uniq
    echo "🔥 Please fix and retry."
    rm $check_output
    exit 1
  fi
  rm $check_output
}