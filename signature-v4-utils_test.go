/*
 * Minio Cloud Storage, (C) 2015, 2016 Minio, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"testing"
)

func TestIsValidRegion(t *testing.T) {
	var regionCases = []struct {
		confRegion string
		reqRegion  string
		expected   bool
	}{
		{"", "US", true},
		{"", "us-east-1", true},
		{"US", "US", true},
		{"US", "us-east-1", true},
		{"us-west-1", "us-west-1", true},
		{"us-west-1", "us-east-1", false},
		{"", "eu-central-1", false},
		{"eu-central-1", "eu-central-1", true},
	}

	for i, test := range regionCases {
		result := isValidRegion(test.reqRegion, test.confRegion)
		if result != test.expected {
			t.Errorf("Failed for test %d with reqRegion = %s confRegion %s", i, test.reqRegion, test.confRegion)
		}
	}
}
