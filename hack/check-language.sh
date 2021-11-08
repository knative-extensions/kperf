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

set +o pipefail

if [ ! -f .wokeignore ]; then
  cat > .wokeignore <<EOF
  vendor/*
  third_party/*
EOF
fi
echo '::group:: Running woke with reviewdog 🐶 ...'
hack/tools/bin/woke --output simple \
  | reviewdog -efm="%f:%l:%c: %m" \
      -name="woke" \
      -filter-mode="nofilter" \
      -fail-on-error="true" \
      -level="error"
