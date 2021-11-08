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

echo '::group:: Running github.com/client9/misspell with reviewdog 🐶 ...'
# Exclude generated and vendored files, plus some legacy
# paths until we update all .gitattributes
git ls-files |
git check-attr --stdin linguist-generated | grep -Ev ': (set|true)$' | cut -d: -f1 |
git check-attr --stdin linguist-vendored | grep -Ev ': (set|true)$' | cut -d: -f1 |
grep -Ev '^(vendor/|third_party/|.git)' |
xargs misspell -i importas -error |
reviewdog -efm="%f:%l:%c: %m" \
      -name="github.com/client9/misspell" \
      -filter-mode="nofilter" \
      -fail-on-error="true" \
      -level="error"
