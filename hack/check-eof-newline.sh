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

# Lint exclude rule:
#  - nothing in vendor/
#  - nothing in third_party
#  - nothing in .git/
#  - no *.ai (Adobe Illustrator) files.
LINT_FILES=$(git ls-files |
git check-attr --stdin linguist-generated | grep -Ev ': (set|true)$' | cut -d: -f1 |
git check-attr --stdin linguist-vendored | grep -Ev ': (set|true)$' | cut -d: -f1 |
grep -Ev '^(vendor/|third_party/|.git)' |
grep -v '\.ai$')
for x in $LINT_FILES; do
  # Based on https://stackoverflow.com/questions/34943632/linux-check-if-there-is-an-empty-line-at-the-end-of-a-file
  if [[ -f $x && ! ( -s "$x" && -z "$(tail -c 1 $x)" ) ]]; then
    # We add 1 to `wc -l` here because of this limitation (from the man page):
    # Characters beyond the final <newline> character will not be included in the line count.
    echo $x:$((1 + $(wc -l $x | tr -s ' ' | cut -d' ' -f 1))): Missing newline
  fi
done |
reviewdog -efm="%f:%l: %m" \
      -name="EOF Newline" \
      -filter-mode="nofilter" \
      -fail-on-error="true" \
      -level="error"
