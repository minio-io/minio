/*
 * MinIO Cloud Storage, (C) 2019 MinIO, Inc.
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

package target

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/minio/minio/pkg/event"
)

// TestDir
var queueDir = filepath.Join(os.TempDir(), "minio_test")

// Sample test event.
var testEvent = event.Event{EventVersion: "1.0", EventSource: "test_source", AwsRegion: "test_region", EventTime: "test_time", EventName: event.ObjectAccessedGet}

// Initialize the store.
func setUpStore(directory string, limit uint16) (Store, error) {
	store := NewQueueStore(queueDir, limit)
	if oErr := store.Open(); oErr != nil {
		return nil, oErr
	}
	return store, nil
}

// Tear down store
func tearDownStore() error {
	if err := os.RemoveAll(queueDir); err != nil {
		return err
	}
	return nil
}

// TestQueueStorePut - tests for store.Put
func TestQueueStorePut(t *testing.T) {
	defer func() {
		if err := tearDownStore(); err != nil {
			t.Fatal("Failed to tear down store ", err)
		}
	}()
	store, err := setUpStore(queueDir, 10000)
	if err != nil {
		t.Fatal("Failed to create a queue store ", err)

	}
	// Put 100 events.
	for i := 0; i < 100; i++ {
		if err := store.Put(testEvent); err != nil {
			t.Fatal("Failed to put to queue store ", err)
		}
	}
	// Count the events.
	if len(store.ListAll()) != 100 {
		t.Fatalf("ListAll() Expected: 100, got %d", len(store.ListAll()))
	}
}

// TestQueueStoreGet - tests for store.Get
func TestQueueStoreGet(t *testing.T) {
	defer func() {
		if err := tearDownStore(); err != nil {
			t.Fatal("Failed to tear down store ", err)
		}
	}()
	store, err := setUpStore(queueDir, 10000)
	if err != nil {
		t.Fatal("Failed to create a queue store ", err)
	}
	// Put 10 events
	for i := 0; i < 10; i++ {
		if err := store.Put(testEvent); err != nil {
			t.Fatal("Failed to put to queue store ", err)
		}
	}
	eventKeys := store.ListAll()
	// Get 10 events.
	if len(eventKeys) == 10 {
		for _, key := range eventKeys {
			event, eErr := store.Get(key)
			if eErr != nil {
				t.Fatal("Failed to Get the event from the queue store ", eErr)
			}
			if !reflect.DeepEqual(testEvent, event) {
				t.Fatalf("Failed to read the event: error: expected = %v, got = %v", testEvent, event)
			}
		}
	} else {
		t.Fatalf("ListAll() Expected: 10, got %d", len(eventKeys))
	}
}

// TestQueueStoreDel - tests for store.Del
func TestQueueStoreDel(t *testing.T) {
	defer func() {
		if err := tearDownStore(); err != nil {
			t.Fatal("Failed to tear down store ", err)
		}
	}()
	store, err := setUpStore(queueDir, 10000)
	if err != nil {
		t.Fatal("Failed to create a queue store ", err)
	}
	// Put 20 events.
	for i := 0; i < 20; i++ {
		if err := store.Put(testEvent); err != nil {
			t.Fatal("Failed to put to queue store ", err)
		}
	}
	eventKeys := store.ListAll()
	// Remove all the events.
	if len(eventKeys) == 20 {
		for _, key := range eventKeys {
			store.Del(key)
		}
	} else {
		t.Fatalf("ListAll() Expected: 20, got %d", len(eventKeys))
	}

	if len(store.ListAll()) != 0 {
		t.Fatalf("ListAll() Expected: 0, got %d", len(store.ListAll()))
	}
}

// TestQueueStoreLimit - tests the event limit for the store.
func TestQueueStoreLimit(t *testing.T) {
	defer func() {
		if err := tearDownStore(); err != nil {
			t.Fatal("Failed to tear down store ", err)
		}
	}()
	// The max limit is set to 5.
	store, err := setUpStore(queueDir, 5)
	if err != nil {
		t.Fatal("Failed to create a queue store ", err)
	}
	for i := 0; i < 5; i++ {
		if err := store.Put(testEvent); err != nil {
			t.Fatal("Failed to put to queue store ", err)
		}
	}
	// Should not allow 6th Put.
	if err := store.Put(testEvent); err == nil {
		t.Fatalf("Expected to fail with %s, but passes", ErrLimitExceeded)
	}
}
