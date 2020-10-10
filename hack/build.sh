#!/bin/bash
set -o pipefail

source_dirs="cmd pkg core"

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

source $(basedir)/hack/build-flags.sh

go_fmt() {
  echo "🧹 ${S}Format"
  find $(echo "${source_dirs}") -name "*.go" -print0 | xargs -0 gofmt -s -w
}

go_build() {
  echo "🚧 Compile"
  go-bindata -nometadata -pkg utils -o ./pkg/command/utils/htmltemplatebindata.go templates/...
  go build -mod=mod -ldflags "$(build_flags $(basedir))" -o kperf ./cmd/...
}

go_test() {
  local red=""
  local reset=""
  # Use color only when a terminal is set
  if [ -t 1 ]; then
    red="[31m"
    reset="[39m"
  fi

  echo "🧪 ${X}Test"
  if ! go test -v ./pkg/...; then
    echo "🔥 ${red}Failure${reset}"
    exit 1
  fi
}

export GO111MODULE=on
export GOPROXY=direct
export GOFLAGS=" -mod=vendor"
go_fmt
go_build

