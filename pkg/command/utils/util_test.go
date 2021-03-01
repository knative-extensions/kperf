// Copyright 2020 The Knative Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

import (
	"bufio"
	"fmt"
	"html/template"
	"os"
	"reflect"
	"strings"
	"testing"

	"bou.ke/monkey"

	"gotest.tools/v3/assert"
)

func TestGenerateCSVFile(t *testing.T) {
	t.Run("generate CSV file successfully", func(t *testing.T) {
		path := "/tmp/generate-csv-test.csv"
		rows := [][]string{{"1", "2", "3"}, {"1", "2", "3"}, {"1", "2", "3"}}
		err := GenerateCSVFile(path, rows)
		assert.NilError(t, err)

		file, err := os.Open(path)
		defer file.Close()
		buffer := make([]byte, 20)
		_, err = file.Read(buffer)
		if err != nil {
			fmt.Print(err)
		}
		assert.NilError(t, err)
	})

	t.Run("return error if path is not available", func(t *testing.T) {
		path := "/tmp"
		rows := [][]string{{"1", "2", "3"}, {"1", "2", "3"}, {"1", "2", "3"}}
		err := GenerateCSVFile(path, rows)
		assert.ErrorContains(t, err, "failed to create csv file open /tmp: is a directory")
	})
}

func TestGenerateHTMLFile(t *testing.T) {
	t.Run("generate HTML file successfully", func(t *testing.T) {
		sourceCSV := "../../../test/asset/test.csv"
		targetHTML := "/tmp/test.html"
		err := GenerateHTMLFile(sourceCSV, targetHTML)
		assert.NilError(t, err)

		file, err := os.Open(targetHTML)
		if err != nil {
			return
		}
		defer file.Close()
		scanner := bufio.NewScanner(file)
		findResult := false
		for scanner.Scan() {
			find := strings.Contains(scanner.Text(), "1, 2, 3\\n4, 5, 6\\n7, 8, 9\\n")
			if find {
				fmt.Println()
				findResult = true
			}
		}
		assert.Equal(t, findResult, true)
	})

	t.Run("failed to read csv file", func(t *testing.T) {
		sourceCSV := "../../../test/asset/test1.csv"
		targetHTML := "/tmp/test.html"
		err := GenerateHTMLFile(sourceCSV, targetHTML)
		assert.ErrorContains(t, err, "failed to read csv file open ../../../test/asset/test1.csv")
	})

	t.Run("failed to parse html template", func(t *testing.T) {
		var tp *template.Template
		monkey.PatchInstanceMethod(reflect.TypeOf(tp), "Parse", func(*template.Template, string) (*template.Template, error) {
			return nil, fmt.Errorf("")
		})
		sourceCSV := "../../../test/asset/test.csv"
		targetHTML := "/tmp/test.html"
		err := GenerateHTMLFile(sourceCSV, targetHTML)
		assert.ErrorContains(t, err, "failed to parse html template")
	})

	t.Run("failed to asset template file", func(t *testing.T) {
		monkey.Patch(Asset, func(name string) ([]byte, error) {
			return []byte{}, fmt.Errorf("")
		})
		sourceCSV := "../../../test/asset/test.csv"
		targetHTML := "/tmp/test.html"
		err := GenerateHTMLFile(sourceCSV, targetHTML)
		assert.ErrorContains(t, err, "failed to load asset")
	})
}
