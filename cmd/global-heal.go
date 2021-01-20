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

package cmd

import (
	"context"
	"sync"
	"time"

	"github.com/minio/minio/cmd/logger"
	"github.com/minio/minio/pkg/color"
	"github.com/minio/minio/pkg/console"
	"github.com/minio/minio/pkg/madmin"
)

const (
	bgHealingUUID = "0000-0000-0000-0000"
)

// NewBgHealSequence creates a background healing sequence
// operation which crawls all objects and heal them.
func newBgHealSequence() *healSequence {
	reqInfo := &logger.ReqInfo{API: "BackgroundHeal"}
	ctx, cancelCtx := context.WithCancel(logger.SetReqInfo(GlobalContext, reqInfo))

	hs := madmin.HealOpts{
		// Remove objects that do not have read-quorum
		Remove:   true,
		ScanMode: madmin.HealNormalScan,
	}

	return &healSequence{
		sourceCh:    make(chan healSource),
		respCh:      make(chan healResult),
		startTime:   UTCNow(),
		clientToken: bgHealingUUID,
		// run-background heal with reserved bucket
		bucket:   minioReservedBucket,
		settings: hs,
		currentStatus: healSequenceStatus{
			Summary:      healNotStartedStatus,
			HealSettings: hs,
		},
		cancelCtx:          cancelCtx,
		ctx:                ctx,
		reportProgress:     false,
		scannedItemsMap:    make(map[madmin.HealItemType]int64),
		healedItemsMap:     make(map[madmin.HealItemType]int64),
		healFailedItemsMap: make(map[string]int64),
	}
}

func getLocalBackgroundHealStatus() (madmin.BgHealState, bool) {
	if globalBackgroundHealState == nil {
		return madmin.BgHealState{}, false
	}

	bgSeq, ok := globalBackgroundHealState.getHealSequenceByToken(bgHealingUUID)
	if !ok {
		return madmin.BgHealState{}, false
	}

	var healDisksMap = map[string]struct{}{}
	for _, ep := range getLocalDisksToHeal() {
		healDisksMap[ep.String()] = struct{}{}
	}

	for _, ep := range globalBackgroundHealState.getHealLocalDisks() {
		if _, ok := healDisksMap[ep.String()]; !ok {
			healDisksMap[ep.String()] = struct{}{}
		}
	}

	var healDisks []string
	for disk := range healDisksMap {
		healDisks = append(healDisks, disk)
	}

	return madmin.BgHealState{
		ScannedItemsCount: bgSeq.getScannedItemsCount(),
		HealDisks:         healDisks,
		HealingDisks:      globalBackgroundHealState.getHealingDisks(),
	}, true
}

func mustGetHealSequence(ctx context.Context) *healSequence {
	// Get background heal sequence to send elements to heal
	for {
		globalHealStateLK.RLock()
		hstate := globalBackgroundHealState
		globalHealStateLK.RUnlock()

		if hstate == nil {
			time.Sleep(time.Second)
			continue
		}

		bgSeq, ok := hstate.getHealSequenceByToken(bgHealingUUID)
		if !ok {
			time.Sleep(time.Second)
			continue
		}
		return bgSeq
	}
}

// healErasureSet lists and heals all objects in a specific erasure set
func healErasureSet(ctx context.Context, buckets []BucketInfo, disks []StorageAPI, tracker *healingTracker) error {
	bgSeq := mustGetHealSequence(ctx)

	buckets = append(buckets, BucketInfo{
		Name: pathJoin(minioMetaBucket, minioConfigPrefix),
	})

	// Try to pro-actively heal backend-encrypted file.
	if err := bgSeq.queueHealTask(healSource{
		bucket: minioMetaBucket,
		object: backendEncryptedFile,
	}, madmin.HealItemMetadata); err != nil {
		if !isErrObjectNotFound(err) && !isErrVersionNotFound(err) {
			logger.LogIf(ctx, err)
		}
	}

	// Reset to where last bucket ended if resuming.
	tracker.resume()

	// Heal all buckets with all objects
	for _, bucket := range buckets {
		if tracker.isHealed(bucket.Name) {
			continue
		}
		tracker.Object = ""
		tracker.Bucket = bucket.Name
		// Heal current bucket
		if err := bgSeq.queueHealTask(healSource{
			bucket: bucket.Name,
		}, madmin.HealItemBucket); err != nil {
			if !isErrObjectNotFound(err) && !isErrVersionNotFound(err) {
				logger.LogIf(ctx, err)
			}
		}

		if serverDebugLog {
			console.Debugf(color.Green("healDisk:")+" healing bucket %s content on erasure set %d\n", bucket.Name, tracker.SetIndex+1)
		}

		var entryChs []FileInfoVersionsCh
		var mu sync.Mutex
		var wg sync.WaitGroup
		for _, disk := range disks {
			disk := disk
			wg.Add(1)
			go func() {
				defer wg.Done()
				entryCh, err := disk.WalkVersions(ctx, bucket.Name, "", "", true, ctx.Done())
				if err != nil {
					// Disk walk returned error, ignore it.
					return
				}
				mu.Lock()
				entryChs = append(entryChs, FileInfoVersionsCh{
					Ch: entryCh,
				})
				mu.Unlock()
			}()
		}
		wg.Wait()

		entriesValid := make([]bool, len(entryChs))
		entries := make([]FileInfoVersions, len(entryChs))

		for {
			entry, _, ok := lexicallySortedEntryVersions(entryChs, entries, entriesValid)
			if !ok {
				break
			}

			for _, version := range entry.Versions {
				if err := bgSeq.queueHealTask(healSource{
					bucket:    bucket.Name,
					object:    version.Name,
					versionID: version.VersionID,
				}, madmin.HealItemObject); err != nil {
					if !isErrObjectNotFound(err) && !isErrVersionNotFound(err) {
						// If not deleted, assume they failed.
						tracker.ObjectsFailed++
						tracker.BytesFailed += uint64(version.Size)
						logger.LogIf(ctx, err)
					}
				} else {
					tracker.ObjectsHealed++
					tracker.BytesDone += uint64(version.Size)
				}
			}
			tracker.Object = entry.Name
			if time.Since(tracker.LastUpdate) > time.Minute {
				logger.LogIf(ctx, tracker.update(ctx))
			}
		}
		select {
		// If context is cancelled don't mark as done...
		case <-ctx.Done():
			return ctx.Err()
		default:
			tracker.bucketDone(bucket.Name)
			logger.LogIf(ctx, tracker.update(ctx))
		}
	}

	return nil
}

// deepHealObject heals given object path in deep to fix bitrot.
func deepHealObject(bucket, object, versionID string) {
	// Get background heal sequence to send elements to heal
	bgSeq, ok := globalBackgroundHealState.getHealSequenceByToken(bgHealingUUID)
	if ok {
		bgSeq.sourceCh <- healSource{
			bucket:    bucket,
			object:    object,
			versionID: versionID,
			opts: &madmin.HealOpts{
				Remove:   true, // if found dangling purge it.
				ScanMode: madmin.HealDeepScan,
			},
		}
	}
}
