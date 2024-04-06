// Copyright (c) 2015-2021 MinIO, Inc.
//
// This file is part of MinIO Object Storage stack
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/minio/madmin-go/v3"
	"github.com/minio/minio/internal/grid"
	"github.com/minio/minio/internal/logger"
	"github.com/minio/pkg/v2/sync/errgroup"
)

//go:generate stringer -type=healingMetric -trimprefix=healingMetric $GOFILE

type healingMetric uint8

const (
	healingMetricBucket healingMetric = iota
	healingMetricObject
	healingMetricCheckAbandonedParts
)

func (er erasureObjects) listAndHeal(bucket, prefix string, scanMode madmin.HealScanMode, healEntry func(string, metaCacheEntry, madmin.HealScanMode) error) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	disks, _ := er.getOnlineDisksWithHealing(false)
	if len(disks) == 0 {
		return errors.New("listAndHeal: No non-healing drives found")
	}

	// How to resolve partial results.
	resolver := metadataResolutionParams{
		dirQuorum: 1,
		objQuorum: 1,
		bucket:    bucket,
		strict:    false, // Allow less strict matching.
	}

	path := baseDirFromPrefix(prefix)
	filterPrefix := strings.Trim(strings.TrimPrefix(prefix, path), slashSeparator)
	if path == prefix {
		filterPrefix = ""
	}

	lopts := listPathRawOptions{
		disks:          disks,
		bucket:         bucket,
		path:           path,
		filterPrefix:   filterPrefix,
		recursive:      true,
		forwardTo:      "",
		minDisks:       1,
		reportNotFound: false,
		agreed: func(entry metaCacheEntry) {
			if err := healEntry(bucket, entry, scanMode); err != nil {
				cancel()
			}
		},
		partial: func(entries metaCacheEntries, _ []error) {
			entry, ok := entries.resolve(&resolver)
			if !ok {
				// check if we can get one entry at least
				// proceed to heal nonetheless.
				entry, _ = entries.firstFound()
			}

			if err := healEntry(bucket, *entry, scanMode); err != nil {
				cancel()
				return
			}
		},
		finished: nil,
	}

	if err := listPathRaw(ctx, lopts); err != nil {
		return fmt.Errorf("listPathRaw returned %w: opts(%#v)", err, lopts)
	}

	return nil
}

// listAllBuckets lists all buckets from all disks. It also
// returns the occurrence of each buckets in all disks
func listAllBuckets(ctx context.Context, storageDisks []StorageAPI, healBuckets map[string]VolInfo, readQuorum int) error {
	g := errgroup.WithNErrs(len(storageDisks))
	var mu sync.Mutex
	for index := range storageDisks {
		index := index
		g.Go(func() error {
			if storageDisks[index] == nil {
				// we ignore disk not found errors
				return nil
			}
			if storageDisks[index].Healing() != nil {
				// we ignore disks under healing
				return nil
			}
			volsInfo, err := storageDisks[index].ListVols(ctx)
			if err != nil {
				return err
			}
			for _, volInfo := range volsInfo {
				// StorageAPI can send volume names which are
				// incompatible with buckets - these are
				// skipped, like the meta-bucket.
				if isReservedOrInvalidBucket(volInfo.Name, false) {
					continue
				}
				mu.Lock()
				if _, ok := healBuckets[volInfo.Name]; !ok {
					healBuckets[volInfo.Name] = volInfo
				}
				mu.Unlock()
			}
			return nil
		}, index)
	}
	return reduceReadQuorumErrs(ctx, g.Wait(), bucketMetadataOpIgnoredErrs, readQuorum)
}

// Only heal on disks where we are sure that healing is needed. We can expand
// this list as and when we figure out more errors can be added to this list safely.
func shouldHealObjectOnDisk(erErr, dataErr error, meta FileInfo, latestMeta FileInfo) bool {
	switch {
	case errors.Is(erErr, errFileNotFound) || errors.Is(erErr, errFileVersionNotFound):
		return true
	case errors.Is(erErr, errFileCorrupt):
		return true
	}
	if erErr == nil {
		if meta.XLV1 {
			// Legacy means heal always
			// always check first.
			return true
		}
		if !meta.Deleted && !meta.IsRemote() {
			// If xl.meta was read fine but there may be problem with the part.N files.
			if IsErr(dataErr, []error{
				errFileNotFound,
				errFileVersionNotFound,
				errFileCorrupt,
			}...) {
				return true
			}
		}
		if !latestMeta.Equals(meta) {
			return true
		}
	}
	return false
}

const (
	xMinIOHealing = ReservedMetadataPrefix + "healing"
	xMinIODataMov = ReservedMetadataPrefix + "data-mov"
)

// SetHealing marks object (version) as being healed.
// Note: this is to be used only from healObject
func (fi *FileInfo) SetHealing() {
	if fi.Metadata == nil {
		fi.Metadata = make(map[string]string)
	}
	fi.Metadata[xMinIOHealing] = "true"
}

// Healing returns true if object is being healed (i.e fi is being passed down
// from healObject)
func (fi FileInfo) Healing() bool {
	_, ok := fi.Metadata[xMinIOHealing]
	return ok
}

// SetDataMov marks object (version) as being currently
// in movement, such as decommissioning or rebalance.
func (fi *FileInfo) SetDataMov() {
	if fi.Metadata == nil {
		fi.Metadata = make(map[string]string)
	}
	fi.Metadata[xMinIODataMov] = "true"
}

// DataMov returns true if object is being in movement
func (fi FileInfo) DataMov() bool {
	_, ok := fi.Metadata[xMinIODataMov]
	return ok
}

func auditHealObject(ctx context.Context, bucket, object, versionID string, result madmin.HealResultItem, err error) {
	if len(logger.AuditTargets()) == 0 {
		return
	}

	opts := AuditLogOptions{
		Event:     "HealObject",
		Bucket:    bucket,
		Object:    object,
		VersionID: versionID,
	}
	if err != nil {
		opts.Error = err.Error()
	}

	if result.After.Drives != nil {
		opts.Tags = map[string]interface{}{"drives-result": result.After.Drives}
	}

	auditLogInternal(ctx, opts)
}

// Heals an object by re-writing corrupt/missing erasure blocks.
func (er *erasureObjects) healObject(ctx context.Context, bucket string, object string, versionID string, opts madmin.HealOpts) (result madmin.HealResultItem, err error) {
	dryRun := opts.DryRun
	scanMode := opts.ScanMode

	storageDisks := er.getDisks()
	storageEndpoints := er.getEndpoints()

	defer func() {
		auditHealObject(ctx, bucket, object, versionID, result, err)
	}()

	if globalTrace.NumSubscribers(madmin.TraceHealing) > 0 {
		startTime := time.Now()
		defer func() {
			healTrace(healingMetricObject, startTime, bucket, object, &opts, err, &result)
		}()
	}
	// Initialize heal result object
	result = madmin.HealResultItem{
		Type:      madmin.HealItemObject,
		Bucket:    bucket,
		Object:    object,
		VersionID: versionID,
		DiskCount: len(storageDisks),
	}

	if !opts.NoLock {
		lk := er.NewNSLock(bucket, object)
		lkctx, err := lk.GetLock(ctx, globalOperationTimeout)
		if err != nil {
			return result, err
		}
		ctx = lkctx.Context()
		defer lk.Unlock(lkctx)
	}

	// Re-read when we have lock...
	partsMetadata, errs := readAllFileInfo(ctx, storageDisks, "", bucket, object, versionID, true, true)
	if isAllNotFound(errs) {
		err := errFileNotFound
		if versionID != "" {
			err = errFileVersionNotFound
		}
		// Nothing to do, file is already gone.
		return er.defaultHealResult(FileInfo{}, storageDisks, storageEndpoints,
			errs, bucket, object, versionID), err
	}

	readQuorum, _, err := objectQuorumFromMeta(ctx, partsMetadata, errs, er.defaultParityCount)
	if err != nil {
		m, err := er.deleteIfDangling(ctx, bucket, object, partsMetadata, errs, nil, ObjectOptions{
			VersionID: versionID,
		})
		errs = make([]error, len(errs))
		for i := range errs {
			errs[i] = err
		}
		if err == nil {
			// Dangling object successfully purged, size is '0'
			m.Size = 0
		}
		// Generate file/version not found with default heal result
		err = errFileNotFound
		if versionID != "" {
			err = errFileVersionNotFound
		}
		return er.defaultHealResult(m, storageDisks, storageEndpoints,
			errs, bucket, object, versionID), err
	}

	result.ParityBlocks = result.DiskCount - readQuorum
	result.DataBlocks = readQuorum

	// List of disks having latest version of the object xl.meta
	// (by modtime).
	onlineDisks, modTime, etag := listOnlineDisks(storageDisks, partsMetadata, errs, readQuorum)

	// Latest FileInfo for reference. If a valid metadata is not
	// present, it is as good as object not found.
	latestMeta, err := pickValidFileInfo(ctx, partsMetadata, modTime, etag, readQuorum)
	if err != nil {
		return result, err
	}

	// List of disks having all parts as per latest metadata.
	// NOTE: do not pass in latestDisks to diskWithAllParts since
	// the diskWithAllParts needs to reach the drive to ensure
	// validity of the metadata content, we should make sure that
	// we pass in disks as is for it to be verified. Once verified
	// the disksWithAllParts() returns the actual disks that can be
	// used here for reconstruction. This is done to ensure that
	// we do not skip drives that have inconsistent metadata to be
	// skipped from purging when they are stale.
	availableDisks, dataErrs, _ := disksWithAllParts(ctx, onlineDisks, partsMetadata,
		errs, latestMeta, bucket, object, scanMode)

	var erasure Erasure
	if !latestMeta.Deleted && !latestMeta.IsRemote() {
		// Initialize erasure coding
		erasure, err = NewErasure(ctx, latestMeta.Erasure.DataBlocks,
			latestMeta.Erasure.ParityBlocks, latestMeta.Erasure.BlockSize)
		if err != nil {
			return result, err
		}
	}

	result.ObjectSize, err = latestMeta.ToObjectInfo(bucket, object, true).GetActualSize()
	if err != nil {
		return result, err
	}

	// Loop to find number of disks with valid data, per-drive
	// data state and a list of outdated disks on which data needs
	// to be healed.
	outDatedDisks := make([]StorageAPI, len(storageDisks))
	disksToHealCount := 0
	for i, v := range availableDisks {
		driveState := ""
		switch {
		case v != nil:
			driveState = madmin.DriveStateOk
		case errs[i] == errDiskNotFound, dataErrs[i] == errDiskNotFound:
			driveState = madmin.DriveStateOffline
		case errs[i] == errFileNotFound, errs[i] == errFileVersionNotFound, errs[i] == errVolumeNotFound:
			fallthrough
		case dataErrs[i] == errFileNotFound, dataErrs[i] == errFileVersionNotFound, dataErrs[i] == errVolumeNotFound:
			driveState = madmin.DriveStateMissing
		default:
			// all remaining cases imply corrupt data/metadata
			driveState = madmin.DriveStateCorrupt
		}

		result.Before.Drives = append(result.Before.Drives, madmin.HealDriveInfo{
			UUID:     "",
			Endpoint: storageEndpoints[i].String(),
			State:    driveState,
		})
		result.After.Drives = append(result.After.Drives, madmin.HealDriveInfo{
			UUID:     "",
			Endpoint: storageEndpoints[i].String(),
			State:    driveState,
		})

		if shouldHealObjectOnDisk(errs[i], dataErrs[i], partsMetadata[i], latestMeta) {
			outDatedDisks[i] = storageDisks[i]
			disksToHealCount++
			continue
		}
	}

	if isAllNotFound(errs) {
		// File is fully gone, fileInfo is empty.
		err := errFileNotFound
		if versionID != "" {
			err = errFileVersionNotFound
		}
		return er.defaultHealResult(FileInfo{}, storageDisks, storageEndpoints, errs,
			bucket, object, versionID), err
	}

	if disksToHealCount == 0 {
		// Nothing to heal!
		return result, nil
	}

	// After this point, only have to repair data on disk - so
	// return if it is a dry-run
	if dryRun {
		return result, nil
	}

	if !latestMeta.XLV1 && !latestMeta.Deleted && disksToHealCount > latestMeta.Erasure.ParityBlocks {
		// Allow for dangling deletes, on versions that have DataDir missing etc.
		// this would end up restoring the correct readable versions.
		m, err := er.deleteIfDangling(ctx, bucket, object, partsMetadata, errs, dataErrs, ObjectOptions{
			VersionID: versionID,
		})
		errs = make([]error, len(errs))
		for i := range errs {
			errs[i] = err
		}
		if err == nil {
			// Dangling object successfully purged, size is '0'
			m.Size = 0
		}
		// Generate file/version not found with default heal result
		err = errFileNotFound
		if versionID != "" {
			err = errFileVersionNotFound
		}
		return er.defaultHealResult(m, storageDisks, storageEndpoints,
			errs, bucket, object, versionID), err
	}

	cleanFileInfo := func(fi FileInfo) FileInfo {
		// Returns a copy of the 'fi' with erasure index, checksums and inline data niled.
		nfi := fi
		if !nfi.IsRemote() {
			nfi.Data = nil
			nfi.Erasure.Index = 0
			nfi.Erasure.Checksums = nil
		}
		return nfi
	}

	// We write at temporary location and then rename to final location.
	tmpID := mustGetUUID()
	migrateDataDir := mustGetUUID()

	// Reorder so that we have data disks first and parity disks next.
	if !latestMeta.Deleted && len(latestMeta.Erasure.Distribution) != len(availableDisks) {
		err := fmt.Errorf("unexpected file distribution (%v) from available disks (%v), looks like backend disks have been manually modified refusing to heal %s/%s(%s)",
			latestMeta.Erasure.Distribution, availableDisks, bucket, object, versionID)
		healingLogOnceIf(ctx, err, "heal-object-available-disks")
		return er.defaultHealResult(latestMeta, storageDisks, storageEndpoints, errs,
			bucket, object, versionID), err
	}

	latestDisks := shuffleDisks(availableDisks, latestMeta.Erasure.Distribution)

	if !latestMeta.Deleted && len(latestMeta.Erasure.Distribution) != len(outDatedDisks) {
		err := fmt.Errorf("unexpected file distribution (%v) from outdated disks (%v), looks like backend disks have been manually modified refusing to heal %s/%s(%s)",
			latestMeta.Erasure.Distribution, outDatedDisks, bucket, object, versionID)
		healingLogOnceIf(ctx, err, "heal-object-outdated-disks")
		return er.defaultHealResult(latestMeta, storageDisks, storageEndpoints, errs,
			bucket, object, versionID), err
	}

	outDatedDisks = shuffleDisks(outDatedDisks, latestMeta.Erasure.Distribution)

	if !latestMeta.Deleted && len(latestMeta.Erasure.Distribution) != len(partsMetadata) {
		err := fmt.Errorf("unexpected file distribution (%v) from metadata entries (%v), looks like backend disks have been manually modified refusing to heal %s/%s(%s)",
			latestMeta.Erasure.Distribution, len(partsMetadata), bucket, object, versionID)
		healingLogOnceIf(ctx, err, "heal-object-metadata-entries")
		return er.defaultHealResult(latestMeta, storageDisks, storageEndpoints, errs,
			bucket, object, versionID), err
	}

	partsMetadata = shufflePartsMetadata(partsMetadata, latestMeta.Erasure.Distribution)

	copyPartsMetadata := make([]FileInfo, len(partsMetadata))
	for i := range latestDisks {
		if latestDisks[i] == nil {
			continue
		}
		copyPartsMetadata[i] = partsMetadata[i]
	}

	for i := range outDatedDisks {
		if outDatedDisks[i] == nil {
			continue
		}
		// Make sure to write the FileInfo information
		// that is expected to be in quorum.
		partsMetadata[i] = cleanFileInfo(latestMeta)
	}

	// source data dir shall be empty in case of XLV1
	// differentiate it with dstDataDir for readability
	// srcDataDir is the one used with newBitrotReader()
	// to read existing content.
	srcDataDir := latestMeta.DataDir
	dstDataDir := latestMeta.DataDir
	if latestMeta.XLV1 {
		dstDataDir = migrateDataDir
	}

	var inlineBuffers []*bytes.Buffer
	if !latestMeta.Deleted && !latestMeta.IsRemote() {
		if latestMeta.InlineData() {
			inlineBuffers = make([]*bytes.Buffer, len(outDatedDisks))
		}

		erasureInfo := latestMeta.Erasure
		for partIndex := 0; partIndex < len(latestMeta.Parts); partIndex++ {
			partSize := latestMeta.Parts[partIndex].Size
			partActualSize := latestMeta.Parts[partIndex].ActualSize
			partModTime := latestMeta.Parts[partIndex].ModTime
			partNumber := latestMeta.Parts[partIndex].Number
			partIdx := latestMeta.Parts[partIndex].Index
			partChecksums := latestMeta.Parts[partIndex].Checksums
			tillOffset := erasure.ShardFileOffset(0, partSize, partSize)
			readers := make([]io.ReaderAt, len(latestDisks))
			prefer := make([]bool, len(latestDisks))
			checksumAlgo := erasureInfo.GetChecksumInfo(partNumber).Algorithm
			for i, disk := range latestDisks {
				if disk == OfflineDisk {
					continue
				}
				checksumInfo := copyPartsMetadata[i].Erasure.GetChecksumInfo(partNumber)
				partPath := pathJoin(object, srcDataDir, fmt.Sprintf("part.%d", partNumber))
				readers[i] = newBitrotReader(disk, copyPartsMetadata[i].Data, bucket, partPath, tillOffset, checksumAlgo,
					checksumInfo.Hash, erasure.ShardSize())
				prefer[i] = disk.Hostname() == ""

			}
			writers := make([]io.Writer, len(outDatedDisks))
			for i, disk := range outDatedDisks {
				if disk == OfflineDisk {
					continue
				}
				partPath := pathJoin(tmpID, dstDataDir, fmt.Sprintf("part.%d", partNumber))
				if len(inlineBuffers) > 0 {
					buf := grid.GetByteBufferCap(int(erasure.ShardFileSize(latestMeta.Size)) + 64)
					inlineBuffers[i] = bytes.NewBuffer(buf[:0])
					defer grid.PutByteBuffer(buf)

					writers[i] = newStreamingBitrotWriterBuffer(inlineBuffers[i], DefaultBitrotAlgorithm, erasure.ShardSize())
				} else {
					writers[i] = newBitrotWriter(disk, bucket, minioMetaTmpBucket, partPath,
						tillOffset, DefaultBitrotAlgorithm, erasure.ShardSize())
				}
			}

			// Heal each part. erasure.Heal() will write the healed
			// part to .minio/tmp/uuid/ which needs to be renamed
			// later to the final location.
			err = erasure.Heal(ctx, writers, readers, partSize, prefer)
			closeBitrotReaders(readers)
			closeBitrotWriters(writers)
			if err != nil {
				return result, err
			}

			// outDatedDisks that had write errors should not be
			// written to for remaining parts, so we nil it out.
			for i, disk := range outDatedDisks {
				if disk == OfflineDisk {
					continue
				}

				// A non-nil stale disk which did not receive
				// a healed part checksum had a write error.
				if writers[i] == nil {
					outDatedDisks[i] = nil
					disksToHealCount--
					continue
				}

				partsMetadata[i].DataDir = dstDataDir
				partsMetadata[i].AddObjectPart(partNumber, "", partSize, partActualSize, partModTime, partIdx, partChecksums)
				if len(inlineBuffers) > 0 && inlineBuffers[i] != nil {
					partsMetadata[i].Data = inlineBuffers[i].Bytes()
					partsMetadata[i].SetInlineData()
				} else {
					partsMetadata[i].Data = nil
				}
			}

			// If all disks are having errors, we give up.
			if disksToHealCount == 0 {
				return result, fmt.Errorf("all drives had write errors, unable to heal %s/%s", bucket, object)
			}

		}

	}

	defer er.deleteAll(context.Background(), minioMetaTmpBucket, tmpID)

	// Rename from tmp location to the actual location.
	for i, disk := range outDatedDisks {
		if disk == OfflineDisk {
			continue
		}

		// record the index of the updated disks
		partsMetadata[i].Erasure.Index = i + 1

		// Attempt a rename now from healed data to final location.
		partsMetadata[i].SetHealing()

		if _, err = disk.RenameData(ctx, minioMetaTmpBucket, tmpID, partsMetadata[i], bucket, object, RenameOptions{}); err != nil {
			return result, err
		}

		// - Remove any remaining parts from outdated disks from before transition.
		if partsMetadata[i].IsRemote() {
			rmDataDir := partsMetadata[i].DataDir
			disk.Delete(ctx, bucket, pathJoin(encodeDirObject(object), rmDataDir), DeleteOptions{
				Immediate: true,
				Recursive: true,
			})
		}

		for i, v := range result.Before.Drives {
			if v.Endpoint == disk.String() {
				result.After.Drives[i].State = madmin.DriveStateOk
			}
		}
	}

	return result, nil
}

// checkAbandonedParts will check if an object has abandoned parts,
// meaning data-dirs or inlined data that are no longer referenced by the xl.meta
// Errors are generally ignored by this function.
func (er *erasureObjects) checkAbandonedParts(ctx context.Context, bucket string, object string, opts madmin.HealOpts) (err error) {
	if !opts.Remove || opts.DryRun {
		return nil
	}
	if globalTrace.NumSubscribers(madmin.TraceHealing) > 0 {
		startTime := time.Now()
		defer func() {
			healTrace(healingMetricCheckAbandonedParts, startTime, bucket, object, nil, err, nil)
		}()
	}
	if !opts.NoLock {
		lk := er.NewNSLock(bucket, object)
		lkctx, err := lk.GetLock(ctx, globalOperationTimeout)
		if err != nil {
			return err
		}
		ctx = lkctx.Context()
		defer lk.Unlock(lkctx)
	}
	var wg sync.WaitGroup
	for _, disk := range er.getDisks() {
		if disk != nil {
			wg.Add(1)
			go func(disk StorageAPI) {
				defer wg.Done()
				_ = disk.CleanAbandonedData(ctx, bucket, object)
			}(disk)
		}
	}
	wg.Wait()
	return nil
}

// healObjectDir - heals object directory specifically, this special call
// is needed since we do not have a special backend format for directories.
func (er *erasureObjects) healObjectDir(ctx context.Context, bucket, object string, dryRun bool, remove bool) (hr madmin.HealResultItem, err error) {
	storageDisks := er.getDisks()
	storageEndpoints := er.getEndpoints()

	// Initialize heal result object
	hr = madmin.HealResultItem{
		Type:         madmin.HealItemObject,
		Bucket:       bucket,
		Object:       object,
		DiskCount:    len(storageDisks),
		ParityBlocks: er.defaultParityCount,
		DataBlocks:   len(storageDisks) - er.defaultParityCount,
		ObjectSize:   0,
	}

	hr.Before.Drives = make([]madmin.HealDriveInfo, len(storageDisks))
	hr.After.Drives = make([]madmin.HealDriveInfo, len(storageDisks))

	errs := statAllDirs(ctx, storageDisks, bucket, object)
	danglingObject := isObjectDirDangling(errs)
	if danglingObject {
		if !dryRun && remove {
			var wg sync.WaitGroup
			// Remove versions in bulk for each disk
			for index, disk := range storageDisks {
				if disk == nil {
					continue
				}
				wg.Add(1)
				go func(index int, disk StorageAPI) {
					defer wg.Done()
					_ = disk.Delete(ctx, bucket, object, DeleteOptions{
						Recursive: false,
						Immediate: false,
					})
				}(index, disk)
			}
			wg.Wait()
		}
	}

	// Prepare object creation in all disks
	for i, err := range errs {
		drive := storageEndpoints[i].String()
		switch err {
		case nil:
			hr.Before.Drives[i] = madmin.HealDriveInfo{Endpoint: drive, State: madmin.DriveStateOk}
			hr.After.Drives[i] = madmin.HealDriveInfo{Endpoint: drive, State: madmin.DriveStateOk}
		case errDiskNotFound:
			hr.Before.Drives[i] = madmin.HealDriveInfo{State: madmin.DriveStateOffline}
			hr.After.Drives[i] = madmin.HealDriveInfo{State: madmin.DriveStateOffline}
		case errVolumeNotFound, errFileNotFound:
			// Bucket or prefix/directory not found
			hr.Before.Drives[i] = madmin.HealDriveInfo{Endpoint: drive, State: madmin.DriveStateMissing}
			hr.After.Drives[i] = madmin.HealDriveInfo{Endpoint: drive, State: madmin.DriveStateMissing}
		default:
			hr.Before.Drives[i] = madmin.HealDriveInfo{Endpoint: drive, State: madmin.DriveStateCorrupt}
			hr.After.Drives[i] = madmin.HealDriveInfo{Endpoint: drive, State: madmin.DriveStateCorrupt}
		}
	}
	if danglingObject || isAllNotFound(errs) {
		// Nothing to do, file is already gone.
		return hr, errFileNotFound
	}

	if dryRun {
		// Quit without try to heal the object dir
		return hr, nil
	}

	for i, err := range errs {
		if err == errVolumeNotFound || err == errFileNotFound {
			// Bucket or prefix/directory not found
			merr := storageDisks[i].MakeVol(ctx, pathJoin(bucket, object))
			switch merr {
			case nil, errVolumeExists:
				hr.After.Drives[i].State = madmin.DriveStateOk
			case errDiskNotFound:
				hr.After.Drives[i].State = madmin.DriveStateOffline
			default:
				hr.After.Drives[i].State = madmin.DriveStateCorrupt
			}
		}
	}
	return hr, nil
}

// Populates default heal result item entries with possible values when we are returning prematurely.
// This is to ensure that in any circumstance we are not returning empty arrays with wrong values.
func (er *erasureObjects) defaultHealResult(lfi FileInfo, storageDisks []StorageAPI, storageEndpoints []Endpoint, errs []error, bucket, object, versionID string) madmin.HealResultItem {
	// Initialize heal result object
	result := madmin.HealResultItem{
		Type:       madmin.HealItemObject,
		Bucket:     bucket,
		Object:     object,
		ObjectSize: lfi.Size,
		VersionID:  versionID,
		DiskCount:  len(storageDisks),
	}

	if lfi.IsValid() {
		result.ParityBlocks = lfi.Erasure.ParityBlocks
	} else {
		// Default to most common configuration for erasure blocks.
		result.ParityBlocks = er.defaultParityCount
	}
	result.DataBlocks = len(storageDisks) - result.ParityBlocks

	for index, disk := range storageDisks {
		if disk == nil {
			result.Before.Drives = append(result.Before.Drives, madmin.HealDriveInfo{
				UUID:     "",
				Endpoint: storageEndpoints[index].String(),
				State:    madmin.DriveStateOffline,
			})
			result.After.Drives = append(result.After.Drives, madmin.HealDriveInfo{
				UUID:     "",
				Endpoint: storageEndpoints[index].String(),
				State:    madmin.DriveStateOffline,
			})
			continue
		}
		driveState := madmin.DriveStateCorrupt
		switch errs[index] {
		case errFileNotFound, errVolumeNotFound:
			driveState = madmin.DriveStateMissing
		case nil:
			driveState = madmin.DriveStateOk
		}
		result.Before.Drives = append(result.Before.Drives, madmin.HealDriveInfo{
			UUID:     "",
			Endpoint: storageEndpoints[index].String(),
			State:    driveState,
		})
		result.After.Drives = append(result.After.Drives, madmin.HealDriveInfo{
			UUID:     "",
			Endpoint: storageEndpoints[index].String(),
			State:    driveState,
		})
	}

	return result
}

// Stat all directories.
func statAllDirs(ctx context.Context, storageDisks []StorageAPI, bucket, prefix string) []error {
	g := errgroup.WithNErrs(len(storageDisks))
	for index, disk := range storageDisks {
		if disk == nil {
			continue
		}
		index := index
		g.Go(func() error {
			entries, err := storageDisks[index].ListDir(ctx, "", bucket, prefix, 1)
			if err != nil {
				return err
			}
			if len(entries) > 0 {
				return errVolumeNotEmpty
			}
			return nil
		}, index)
	}

	return g.Wait()
}

func isAllVolumeNotFound(errs []error) bool {
	return countErrs(errs, errVolumeNotFound) == len(errs)
}

// isAllNotFound will return if any element of the error slice is not
// errFileNotFound, errFileVersionNotFound or errVolumeNotFound.
// A 0 length slice will always return false.
func isAllNotFound(errs []error) bool {
	for _, err := range errs {
		if err != nil {
			switch err.Error() {
			case errFileNotFound.Error():
				fallthrough
			case errVolumeNotFound.Error():
				fallthrough
			case errFileVersionNotFound.Error():
				continue
			}
		}
		return false
	}
	return len(errs) > 0
}

// isAllBucketsNotFound will return true if all the errors are either errFileNotFound
// or errFileCorrupt
// A 0 length slice will always return false.
func isAllBucketsNotFound(errs []error) bool {
	if len(errs) == 0 {
		return false
	}
	notFoundCount := 0
	for _, err := range errs {
		if err != nil {
			if errors.Is(err, errVolumeNotFound) {
				notFoundCount++
			} else if isErrBucketNotFound(err) {
				notFoundCount++
			}
		}
	}
	return len(errs) == notFoundCount
}

// ObjectDir is considered dangling/corrupted if any only
// if total disks - a combination of corrupted and missing
// files is lesser than N/2+1 number of disks.
// If no files were found false will be returned.
func isObjectDirDangling(errs []error) (ok bool) {
	var found int
	var notFound int
	var foundNotEmpty int
	var otherFound int
	for _, readErr := range errs {
		switch {
		case readErr == nil:
			found++
		case readErr == errFileNotFound || readErr == errVolumeNotFound:
			notFound++
		case readErr == errVolumeNotEmpty:
			foundNotEmpty++
		default:
			otherFound++
		}
	}
	found = found + foundNotEmpty + otherFound
	return found < notFound && found > 0
}

// Object is considered dangling/corrupted if and only
// if total disks - a combination of corrupted and missing
// files is lesser than number of data blocks.
func isObjectDangling(metaArr []FileInfo, errs []error, dataErrs []error) (validMeta FileInfo, ok bool) {
	// We can consider an object data not reliable
	// when xl.meta is not found in read quorum disks.
	// or when xl.meta is not readable in read quorum disks.
	danglingErrsCount := func(cerrs []error) (int, int) {
		var (
			notFoundCount      int
			nonActionableCount int
		)
		for _, readErr := range cerrs {
			if readErr == nil {
				continue
			}
			switch {
			case errors.Is(readErr, errFileNotFound) || errors.Is(readErr, errFileVersionNotFound):
				notFoundCount++
			default:
				// All other errors are non-actionable
				nonActionableCount++
			}
		}
		return notFoundCount, nonActionableCount
	}

	notFoundMetaErrs, nonActionableMetaErrs := danglingErrsCount(errs)
	notFoundPartsErrs, nonActionablePartsErrs := danglingErrsCount(dataErrs)

	for _, m := range metaArr {
		if m.IsValid() {
			validMeta = m
			break
		}
	}

	if !validMeta.IsValid() {
		// validMeta is invalid because all xl.meta is missing apparently
		// we should figure out if dataDirs are also missing > dataBlocks.
		dataBlocks := (len(dataErrs) + 1) / 2
		if notFoundPartsErrs > dataBlocks {
			// Not using parity to ensure that we do not delete
			// any valid content, if any is recoverable. But if
			// notFoundDataDirs are already greater than the data
			// blocks all bets are off and it is safe to purge.
			//
			// This is purely a defensive code, ideally parityBlocks
			// is sufficient, however we can't know that since we
			// do have the FileInfo{}.
			return validMeta, true
		}

		// We have no idea what this file is, leave it as is.
		return validMeta, false
	}

	if nonActionableMetaErrs > 0 || nonActionablePartsErrs > 0 {
		return validMeta, false
	}

	if validMeta.Deleted {
		// notFoundPartsErrs is ignored since
		// - delete marker does not have any parts
		dataBlocks := (len(errs) + 1) / 2
		return validMeta, notFoundMetaErrs > dataBlocks
	}

	// TODO: It is possible to replay the object via just single
	// xl.meta file, considering quorum number of data-dirs are still
	// present on other drives.
	//
	// However this requires a bit of a rewrite, leave this up for
	// future work.
	if notFoundMetaErrs > 0 && notFoundMetaErrs > validMeta.Erasure.ParityBlocks {
		// All xl.meta is beyond data blocks missing, this is dangling
		return validMeta, true
	}

	if !validMeta.IsRemote() && notFoundPartsErrs > 0 && notFoundPartsErrs > validMeta.Erasure.ParityBlocks {
		// All data-dir is beyond data blocks missing, this is dangling
		return validMeta, true
	}

	return validMeta, false
}

// HealObject - heal the given object, automatically deletes the object if stale/corrupted if `remove` is true.
func (er erasureObjects) HealObject(ctx context.Context, bucket, object, versionID string, opts madmin.HealOpts) (hr madmin.HealResultItem, err error) {
	// Create context that also contains information about the object and bucket.
	// The top level handler might not have this information.
	reqInfo := logger.GetReqInfo(ctx)
	var newReqInfo *logger.ReqInfo
	if reqInfo != nil {
		newReqInfo = logger.NewReqInfo(reqInfo.RemoteHost, reqInfo.UserAgent, reqInfo.DeploymentID, reqInfo.RequestID, reqInfo.API, bucket, object)
	} else {
		newReqInfo = logger.NewReqInfo("", "", globalDeploymentID(), "", "Heal", bucket, object)
	}
	healCtx := logger.SetReqInfo(GlobalContext, newReqInfo)

	// Healing directories handle it separately.
	if HasSuffix(object, SlashSeparator) {
		hr, err := er.healObjectDir(healCtx, bucket, object, opts.DryRun, opts.Remove)
		return hr, toObjectErr(err, bucket, object)
	}

	storageDisks := er.getDisks()
	storageEndpoints := er.getEndpoints()

	// When versionID is empty, we read directly from the `null` versionID for healing.
	if versionID == "" {
		versionID = nullVersionID
	}

	// Perform quick read without lock.
	// This allows to quickly check if all is ok or all are missing.
	_, errs := readAllFileInfo(healCtx, storageDisks, "", bucket, object, versionID, false, false)
	if isAllNotFound(errs) {
		err := errFileNotFound
		if versionID != "" {
			err = errFileVersionNotFound
		}
		// Nothing to do, file is already gone.
		return er.defaultHealResult(FileInfo{}, storageDisks, storageEndpoints,
			errs, bucket, object, versionID), toObjectErr(err, bucket, object, versionID)
	}

	// Heal the object.
	hr, err = er.healObject(healCtx, bucket, object, versionID, opts)
	if errors.Is(err, errFileCorrupt) && opts.ScanMode != madmin.HealDeepScan {
		// Instead of returning an error when a bitrot error is detected
		// during a normal heal scan, heal again with bitrot flag enabled.
		opts.ScanMode = madmin.HealDeepScan
		hr, err = er.healObject(healCtx, bucket, object, versionID, opts)
	}
	return hr, toObjectErr(err, bucket, object, versionID)
}

// healTrace sends healing results to trace output.
func healTrace(funcName healingMetric, startTime time.Time, bucket, object string, opts *madmin.HealOpts, err error, result *madmin.HealResultItem) {
	tr := madmin.TraceInfo{
		TraceType: madmin.TraceHealing,
		Time:      startTime,
		NodeName:  globalLocalNodeName,
		FuncName:  "heal." + funcName.String(),
		Duration:  time.Since(startTime),
		Path:      pathJoin(bucket, decodeDirObject(object)),
	}
	if opts != nil {
		tr.Custom = map[string]string{
			"dry":    fmt.Sprint(opts.DryRun),
			"remove": fmt.Sprint(opts.Remove),
			"mode":   fmt.Sprint(opts.ScanMode),
		}
		if result != nil {
			tr.Custom["version-id"] = result.VersionID
			tr.Custom["disks"] = strconv.Itoa(result.DiskCount)
		}
	}
	if err != nil {
		tr.Error = err.Error()
	} else {
		tr.HealResult = result
	}
	globalTrace.Publish(tr)
}
