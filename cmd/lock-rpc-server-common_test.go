/*
 * Minio Cloud Storage, (C) 2016, 2017 Minio, Inc.
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

package cmd

import (
	"os"
	"reflect"
	"testing"
	"time"
)

// Test function to remove lock entries from map only in case they still exist based on name & uid combination
func TestLockRpcServerRemoveEntryIfExists(t *testing.T) {
	testPath, locker, _ := createLockTestServer(t)
	defer os.RemoveAll(testPath)

	lri := lockRequesterInfo{
		Writer:          false,
		Node:            "host",
		ServiceEndpoint: "rpc-path",
		UID:             "0123-4567",
		Timestamp:       UTCNow(),
		TimeLastCheck:   UTCNow(),
	}
	nlrip := nameLockRequesterInfoPair{name: "name", lri: lri}

	// first test by simulating item has already been deleted
	locker.ll.removeEntryIfExists(nlrip)
	{
		gotLri, _ := locker.ll.lockMap["name"]
		expectedLri := []lockRequesterInfo(nil)
		if !reflect.DeepEqual(expectedLri, gotLri) {
			t.Errorf("Expected %#v, got %#v", expectedLri, gotLri)
		}
	}

	// then test normal deletion
	locker.ll.lockMap["name"] = []lockRequesterInfo{lri} // add item
	locker.ll.removeEntryIfExists(nlrip)
	{
		gotLri, _ := locker.ll.lockMap["name"]
		expectedLri := []lockRequesterInfo(nil)
		if !reflect.DeepEqual(expectedLri, gotLri) {
			t.Errorf("Expected %#v, got %#v", expectedLri, gotLri)
		}
	}
}

// Test function to remove lock entries from map based on name & uid combination
func TestLockRpcServerRemoveEntry(t *testing.T) {
	testPath, locker, _ := createLockTestServer(t)
	defer os.RemoveAll(testPath)

	lockRequesterInfo1 := lockRequesterInfo{
		Writer:          true,
		Node:            "host",
		ServiceEndpoint: "rpc-path",
		UID:             "0123-4567",
		Timestamp:       UTCNow(),
		TimeLastCheck:   UTCNow(),
	}
	lockRequesterInfo2 := lockRequesterInfo{
		Writer:          true,
		Node:            "host",
		ServiceEndpoint: "rpc-path",
		UID:             "89ab-cdef",
		Timestamp:       UTCNow(),
		TimeLastCheck:   UTCNow(),
	}

	locker.ll.lockMap["name"] = []lockRequesterInfo{
		lockRequesterInfo1,
		lockRequesterInfo2,
	}

	lri, _ := locker.ll.lockMap["name"]

	// test unknown uid
	if locker.ll.removeEntry("name", "unknown-uid", &lri) {
		t.Errorf("Expected %#v, got %#v", false, true)
	}

	if !locker.ll.removeEntry("name", "0123-4567", &lri) {
		t.Errorf("Expected %#v, got %#v", true, false)
	} else {
		gotLri, _ := locker.ll.lockMap["name"]
		expectedLri := []lockRequesterInfo{lockRequesterInfo2}
		if !reflect.DeepEqual(expectedLri, gotLri) {
			t.Errorf("Expected %#v, got %#v", expectedLri, gotLri)
		}
	}

	if !locker.ll.removeEntry("name", "89ab-cdef", &lri) {
		t.Errorf("Expected %#v, got %#v", true, false)
	} else {
		gotLri, _ := locker.ll.lockMap["name"]
		expectedLri := []lockRequesterInfo(nil)
		if !reflect.DeepEqual(expectedLri, gotLri) {
			t.Errorf("Expected %#v, got %#v", expectedLri, gotLri)
		}
	}
}

// Tests function returning long lived locks.
func TestLockRpcServerGetLongLivedLocks(t *testing.T) {
	ut := UTCNow()
	// Collection of test cases for verifying returning valid long lived locks.
	testCases := []struct {
		lockMap      map[string][]lockRequesterInfo
		lockInterval time.Duration
		expectedNSLR []nameLockRequesterInfoPair
	}{
		// Testcase - 1 validates long lived locks, returns empty list.
		{
			lockMap: map[string][]lockRequesterInfo{
				"test": {{
					Writer:          true,
					Node:            "10.1.10.21",
					ServiceEndpoint: "/lock/mnt/disk1",
					UID:             "10000112",
					Timestamp:       ut,
					TimeLastCheck:   ut,
				}},
			},
			lockInterval: 1 * time.Minute,
			expectedNSLR: []nameLockRequesterInfoPair{},
		},
		// Testcase - 2 validates long lived locks, returns at least one list.
		{
			lockMap: map[string][]lockRequesterInfo{
				"test": {{
					Writer:          true,
					Node:            "10.1.10.21",
					ServiceEndpoint: "/lock/mnt/disk1",
					UID:             "10000112",
					Timestamp:       ut,
					TimeLastCheck:   ut.Add(-2 * time.Minute),
				}},
			},
			lockInterval: 1 * time.Minute,
			expectedNSLR: []nameLockRequesterInfoPair{
				{
					name: "test",
					lri: lockRequesterInfo{
						Writer:          true,
						Node:            "10.1.10.21",
						ServiceEndpoint: "/lock/mnt/disk1",
						UID:             "10000112",
						Timestamp:       ut,
						TimeLastCheck:   ut.Add(-2 * time.Minute),
					},
				},
			},
		},
	}
	// Validates all test cases here.
	for i, testCase := range testCases {
		nsLR := getLongLivedLocks(testCase.lockMap, testCase.lockInterval)
		if !reflect.DeepEqual(testCase.expectedNSLR, nsLR) {
			t.Errorf("Test %d: Expected %#v, got %#v", i+1, testCase.expectedNSLR, nsLR)
		}
	}
}
