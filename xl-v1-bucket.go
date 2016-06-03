/*
 * Minio Cloud Storage, (C) 2016 Minio, Inc.
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
	"sort"
	"sync"
)

/// Bucket operations

// MakeBucket - make a bucket.
func (xl xlObjects) MakeBucket(bucket string) error {
	// Verify if bucket is valid.
	if !IsValidBucketName(bucket) {
		return BucketNameInvalid{Bucket: bucket}
	}

	nsMutex.Lock(bucket, "")
	defer nsMutex.Unlock(bucket, "")

	// Err counters.
	createVolErr := 0       // Count generic create vol errs.
	volumeExistsErrCnt := 0 // Count all errVolumeExists errs.

	// Initialize sync waitgroup.
	var wg = &sync.WaitGroup{}

	// Initialize list of errors.
	var dErrs = make([]error, len(xl.storageDisks))

	// Make a volume entry on all underlying storage disks.
	for index, disk := range xl.storageDisks {
		if disk == nil {
			dErrs[index] = errDiskNotFound
			continue
		}
		wg.Add(1)
		// Make a volume inside a go-routine.
		go func(index int, disk StorageAPI) {
			defer wg.Done()
			err := disk.MakeVol(bucket)
			if err != nil {
				dErrs[index] = err
				return
			}
			dErrs[index] = nil
		}(index, disk)
	}

	// Wait for all make vol to finish.
	wg.Wait()

	// Look for specific errors and count them to be verified later.
	for _, err := range dErrs {
		if err == nil {
			continue
		}
		// if volume already exists, count them.
		if err == errVolumeExists {
			volumeExistsErrCnt++
			continue
		}

		// Update error counter separately.
		createVolErr++
	}

	// Return err if all disks report volume exists.
	if volumeExistsErrCnt > len(xl.storageDisks)-xl.readQuorum {
		return toObjectErr(errVolumeExists, bucket)
	} else if createVolErr > len(xl.storageDisks)-xl.writeQuorum {
		// Return errXLWriteQuorum if errors were more than allowed write quorum.
		return toObjectErr(errXLWriteQuorum, bucket)
	}
	return nil
}

// getBucketInfo - returns the BucketInfo from one of the load balanced disks.
func (xl xlObjects) getBucketInfo(bucketName string) (bucketInfo BucketInfo, err error) {
	for _, disk := range xl.getLoadBalancedQuorumDisks() {
		if disk == nil {
			continue
		}
		var volInfo VolInfo
		volInfo, err = disk.StatVol(bucketName)
		if err != nil {
			// For some reason disk went offline pick the next one.
			if err == errDiskNotFound {
				continue
			}
			return BucketInfo{}, err
		}
		bucketInfo = BucketInfo{
			Name:    volInfo.Name,
			Created: volInfo.Created,
		}
		break
	}
	return bucketInfo, nil
}

// Checks whether bucket exists.
func (xl xlObjects) isBucketExist(bucket string) bool {
	nsMutex.RLock(bucket, "")
	defer nsMutex.RUnlock(bucket, "")

	// Check whether bucket exists.
	_, err := xl.getBucketInfo(bucket)
	if err != nil {
		if err == errVolumeNotFound {
			return false
		}
		errorIf(err, "Stat failed on bucket "+bucket+".")
		return false
	}
	return true
}

// GetBucketInfo - returns BucketInfo for a bucket.
func (xl xlObjects) GetBucketInfo(bucket string) (BucketInfo, error) {
	// Verify if bucket is valid.
	if !IsValidBucketName(bucket) {
		return BucketInfo{}, BucketNameInvalid{Bucket: bucket}
	}
	nsMutex.RLock(bucket, "")
	defer nsMutex.RUnlock(bucket, "")
	bucketInfo, err := xl.getBucketInfo(bucket)
	if err != nil {
		return BucketInfo{}, toObjectErr(err, bucket)
	}
	return bucketInfo, nil
}

// listBuckets - returns list of all buckets from a disk picked at random.
func (xl xlObjects) listBuckets() (bucketsInfo []BucketInfo, err error) {
	for _, disk := range xl.getLoadBalancedQuorumDisks() {
		if disk == nil {
			continue
		}
		var volsInfo []VolInfo
		volsInfo, err = disk.ListVols()
		// Ignore any disks not found.
		if err == errDiskNotFound {
			continue
		}
		if err == nil {
			// NOTE: The assumption here is that volumes across all disks in
			// readQuorum have consistent view i.e they all have same number
			// of buckets. This is essentially not verified since healing
			// should take care of this.
			var bucketsInfo []BucketInfo
			for _, volInfo := range volsInfo {
				// StorageAPI can send volume names which are incompatible
				// with buckets, handle it and skip them.
				if !IsValidBucketName(volInfo.Name) {
					continue
				}
				bucketsInfo = append(bucketsInfo, BucketInfo{
					Name:    volInfo.Name,
					Created: volInfo.Created,
				})
			}
			return bucketsInfo, nil
		}
		break
	}
	return nil, err
}

// ListBuckets - lists all the buckets, sorted by its name.
func (xl xlObjects) ListBuckets() ([]BucketInfo, error) {
	bucketInfos, err := xl.listBuckets()
	if err != nil {
		return nil, toObjectErr(err)
	}
	// Sort by bucket name before returning.
	sort.Sort(byBucketName(bucketInfos))
	return bucketInfos, nil
}

// DeleteBucket - deletes a bucket.
func (xl xlObjects) DeleteBucket(bucket string) error {
	// Verify if bucket is valid.
	if !IsValidBucketName(bucket) {
		return BucketNameInvalid{Bucket: bucket}
	}

	nsMutex.Lock(bucket, "")
	defer nsMutex.Unlock(bucket, "")

	// Collect if all disks report volume not found.
	var volumeNotFoundErrCnt int

	var wg = &sync.WaitGroup{}
	var dErrs = make([]error, len(xl.storageDisks))

	// Remove a volume entry on all underlying storage disks.
	for index, disk := range xl.storageDisks {
		if disk == nil {
			dErrs[index] = errDiskNotFound
			continue
		}
		wg.Add(1)
		// Delete volume inside a go-routine.
		go func(index int, disk StorageAPI) {
			defer wg.Done()
			err := disk.DeleteVol(bucket)
			if err != nil {
				dErrs[index] = err
			}
		}(index, disk)
	}

	// Wait for all the delete vols to finish.
	wg.Wait()

	// Count the errors for known errors, return quickly if we found
	// an unknown error.
	for _, err := range dErrs {
		if err != nil {
			// We ignore error if errVolumeNotFound or errDiskNotFound
			if err == errVolumeNotFound || err == errDiskNotFound {
				volumeNotFoundErrCnt++
				continue
			}
			return toObjectErr(err, bucket)
		}
	}

	// Return errVolumeNotFound if all disks report volume not found.
	if volumeNotFoundErrCnt == len(xl.storageDisks) {
		return toObjectErr(errVolumeNotFound, bucket)
	}

	return nil
}
