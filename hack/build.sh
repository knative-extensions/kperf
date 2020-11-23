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

check_license() {
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

export GO111MODULE=on
export GOPROXY=direct
export GOFLAGS=" -mod=vendor"
check_license
go_fmt
go_build

