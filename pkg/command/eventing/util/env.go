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

package util

import (
	"fmt"
	"os"
	"strconv"
)

//TODO use existing util pkpg
func GetEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func GetEnvInt(key, fallback string) int {
	value := GetEnv(key, fallback)
	v, err := strconv.Atoi(value)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not get int value out of %s for environment variable %s error: %v\n", value, key, err)
		os.Exit(1)
	}
	return v
}

func GetEnvFloat64(key, fallback string) float64 {
	value := GetEnv(key, fallback)
	v, err := strconv.ParseFloat(value, 64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not get int value out of %s for environment variable %s error: %v\n", value, key, err)
		os.Exit(1)
	}
	return v
}
