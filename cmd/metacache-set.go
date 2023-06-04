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
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/minio/minio/internal/bucket/lifecycle"
	"github.com/minio/minio/internal/bucket/object/lock"
	"github.com/minio/minio/internal/bucket/versioning"
	"github.com/minio/minio/internal/color"
	"github.com/minio/minio/internal/hash"
	"github.com/minio/minio/internal/logger"
	"github.com/minio/pkg/console"
)

type listPathOptions struct {

	// Replication configuration
	Replication replicationConfig

	// Versioning config is used for if the path
	// has versioning enabled.
	Versioning *versioning.Versioning

	// Lifecycle performs filtering based on lifecycle.
	// This will filter out objects if the most recent version should be deleted by lifecycle.
	// Is not transferred across request calls.
	Lifecycle *lifecycle.Lifecycle

	// Marker to resume listing.
	// The response will be the first entry >= this object name.
	Marker string

	// Scan/return only content with prefix.
	Prefix string

	// ID of the listing.
	// This will be used to persist the list.
	ID string

	// The number of disks to ask.
	AskDisks string

	// FilterPrefix will return only results with this prefix when scanning.
	// Should never contain a slash.
	// Prefix should still be set.
	FilterPrefix string

	// Separator to use.
	Separator string

	// Bucket of the listing.
	Bucket string

	// Directory inside the bucket.
	// When unset listPath will set this based on Prefix
	BaseDir string

	// Retention configuration, needed to be passed along with lifecycle if set.
	Retention lock.Retention

	// Limit the number of results.
	Limit int

	// pool and set of where the cache is located.
	pool, set int

	// InclDeleted will keep all entries where latest version is a delete marker.
	InclDeleted bool

	// Versioned is this a ListObjectVersions call.
	Versioned bool

	// Transient is set if the cache is transient due to an error or being a reserved bucket.
	// This means the cache metadata will not be persisted on disk.
	// A transient result will never be returned from the cache so knowing the list id is required.
	Transient bool

	// Include pure directories.
	IncludeDirectories bool

	// StopDiskAtLimit will stop listing on each disk when limit number off objects has been returned.
	StopDiskAtLimit bool

	// Create indicates that the lister should not attempt to load an existing cache.
	Create bool

	// Scan recursively.
	// If false only main directory will be scanned.
	// Should always be true if Separator is n SlashSeparator.
	Recursive bool
}

func init() {
	gob.Register(listPathOptions{})
}

func (o *listPathOptions) setBucketMeta(ctx context.Context) {
	lc, _ := globalLifecycleSys.Get(o.Bucket)
	vc, _ := globalBucketVersioningSys.Get(o.Bucket)

	// Check if bucket is object locked.
	rcfg, _ := globalBucketObjectLockSys.Get(o.Bucket)
	replCfg, _, _ := globalBucketMetadataSys.GetReplicationConfig(ctx, o.Bucket)
	tgts, _ := globalBucketTargetSys.ListBucketTargets(ctx, o.Bucket)
	o.Lifecycle = lc
	o.Versioning = vc
	o.Replication = replicationConfig{
		Config:  replCfg,
		remotes: tgts,
	}
	o.Retention = rcfg
}

// newMetacache constructs a new metacache from the options.
func (o listPathOptions) newMetacache() metacache {
	return metacache{
		id:          o.ID,
		bucket:      o.Bucket,
		root:        o.BaseDir,
		recursive:   o.Recursive,
		status:      scanStateStarted,
		error:       "",
		started:     UTCNow(),
		lastHandout: UTCNow(),
		lastUpdate:  UTCNow(),
		ended:       time.Time{},
		dataVersion: metacacheStreamVersion,
		filter:      o.FilterPrefix,
	}
}

func (o *listPathOptions) debugf(format string, data ...interface{}) {
	if serverDebugLog {
		console.Debugf(format+"\n", data...)
	}
}

func (o *listPathOptions) debugln(data ...interface{}) {
	if serverDebugLog {
		console.Debugln(data...)
	}
}

// gatherResults will collect all results on the input channel and filter results according to the options.
// Caller should close the channel when done.
// The returned function will return the results once there is enough or input is closed,
// or the context is canceled.
func (o *listPathOptions) gatherResults(ctx context.Context, in <-chan metaCacheEntry) func() (metaCacheEntriesSorted, error) {
	resultsDone := make(chan metaCacheEntriesSorted)
	// Copy so we can mutate
	resCh := resultsDone
	var done bool
	var mu sync.Mutex
	resErr := io.EOF

	go func() {
		var results metaCacheEntriesSorted
		var returned bool
		for entry := range in {
			if returned {
				// past limit
				continue
			}
			mu.Lock()
			returned = done
			mu.Unlock()
			if returned {
				resCh = nil
				continue
			}
			if !o.IncludeDirectories && (entry.isDir() || (!o.Versioned && entry.isObjectDir() && entry.isLatestDeletemarker())) {
				continue
			}
			if o.Marker != "" && entry.name < o.Marker {
				continue
			}
			if !strings.HasPrefix(entry.name, o.Prefix) {
				continue
			}
			if !o.Recursive && !entry.isInDir(o.Prefix, o.Separator) {
				continue
			}
			if !o.InclDeleted && entry.isObject() && entry.isLatestDeletemarker() && !entry.isObjectDir() {
				continue
			}
			if o.Limit > 0 && results.len() >= o.Limit {
				// We have enough and we have more.
				// Do not return io.EOF
				if resCh != nil {
					resErr = nil
					resCh <- results
					resCh = nil
					returned = true
				}
				continue
			}
			results.o = append(results.o, entry)
		}
		if resCh != nil {
			resErr = io.EOF
			select {
			case <-ctx.Done():
				// Nobody wants it.
			case resCh <- results:
			}
		}
	}()
	return func() (metaCacheEntriesSorted, error) {
		select {
		case <-ctx.Done():
			mu.Lock()
			done = true
			mu.Unlock()
			return metaCacheEntriesSorted{}, ctx.Err()
		case r := <-resultsDone:
			return r, resErr
		}
	}
}

// findFirstPart will find the part with 0 being the first that corresponds to the marker in the options.
// io.ErrUnexpectedEOF is returned if the place containing the marker hasn't been scanned yet.
// io.EOF indicates the marker is beyond the end of the stream and does not exist.
func (o *listPathOptions) findFirstPart(fi FileInfo) (int, error) {
	search := o.Marker
	if search == "" {
		search = o.Prefix
	}
	if search == "" {
		return 0, nil
	}
	o.debugln("searching for ", search)
	var tmp metacacheBlock
	json := jsoniter.ConfigCompatibleWithStandardLibrary
	i := 0
	for {
		partKey := fmt.Sprintf("%s-metacache-part-%d", ReservedMetadataPrefixLower, i)
		v, ok := fi.Metadata[partKey]
		if !ok {
			o.debugln("no match in metadata, waiting")
			return -1, io.ErrUnexpectedEOF
		}
		err := json.Unmarshal([]byte(v), &tmp)
		if !ok {
			logger.LogIf(context.Background(), err)
			return -1, err
		}
		if tmp.First == "" && tmp.Last == "" && tmp.EOS {
			return 0, errFileNotFound
		}
		if tmp.First >= search {
			o.debugln("First >= search", v)
			return i, nil
		}
		if tmp.Last >= search {
			o.debugln("Last >= search", v)
			return i, nil
		}
		if tmp.EOS {
			o.debugln("no match, at EOS", v)
			return -3, io.EOF
		}
		o.debugln("First ", tmp.First, "<", search, " search", i)
		i++
	}
}

// updateMetacacheListing will update the metacache listing.
func (o *listPathOptions) updateMetacacheListing(m metacache, rpc *peerRESTClient) (metacache, error) {
	if rpc == nil {
		return localMetacacheMgr.updateCacheEntry(m)
	}
	return rpc.UpdateMetacacheListing(context.Background(), m)
}

func getMetacacheBlockInfo(fi FileInfo, block int) (*metacacheBlock, error) {
	var tmp metacacheBlock
	partKey := fmt.Sprintf("%s-metacache-part-%d", ReservedMetadataPrefixLower, block)
	v, ok := fi.Metadata[partKey]
	if !ok {
		return nil, io.ErrUnexpectedEOF
	}
	return &tmp, json.Unmarshal([]byte(v), &tmp)
}

const metacachePrefix = ".metacache"

func metacachePrefixForID(bucket, id string) string {
	return pathJoin(bucketMetaPrefix, bucket, metacachePrefix, id)
}

// objectPath returns the object path of the cache.
func (o *listPathOptions) objectPath(block int) string {
	return pathJoin(metacachePrefixForID(o.Bucket, o.ID), "block-"+strconv.Itoa(block)+".s2")
}

func (o *listPathOptions) SetFilter() {
	switch {
	case metacacheSharePrefix:
		return
	case o.Prefix == o.BaseDir:
		// No additional prefix
		return
	}
	// Remove basedir.
	o.FilterPrefix = strings.TrimPrefix(o.Prefix, o.BaseDir)
	// Remove leading and trailing slashes.
	o.FilterPrefix = strings.Trim(o.FilterPrefix, slashSeparator)

	if strings.Contains(o.FilterPrefix, slashSeparator) {
		// Sanity check, should not happen.
		o.FilterPrefix = ""
	}
}

// filter will apply the options and return the number of objects requested by the limit.
// Will return io.EOF if there are no more entries with the same filter.
// The last entry can be used as a marker to resume the listing.
func (r *metacacheReader) filter(o listPathOptions) (entries metaCacheEntriesSorted, err error) {
	// Forward to prefix, if any
	err = r.forwardTo(o.Prefix)
	if err != nil {
		return entries, err
	}
	if o.Marker != "" {
		err = r.forwardTo(o.Marker)
		if err != nil {
			return entries, err
		}
	}
	o.debugln("forwarded to ", o.Prefix, "marker:", o.Marker, "sep:", o.Separator)

	// Filter
	if !o.Recursive {
		entries.o = make(metaCacheEntries, 0, o.Limit)
		pastPrefix := false
		err := r.readFn(func(entry metaCacheEntry) bool {
			if o.Prefix != "" && !strings.HasPrefix(entry.name, o.Prefix) {
				// We are past the prefix, don't continue.
				pastPrefix = true
				return false
			}
			if !o.IncludeDirectories && (entry.isDir() || (!o.Versioned && entry.isObjectDir() && entry.isLatestDeletemarker())) {
				return true
			}
			if !entry.isInDir(o.Prefix, o.Separator) {
				return true
			}
			if !o.InclDeleted && entry.isObject() && entry.isLatestDeletemarker() && !entry.isObjectDir() {
				return true
			}
			if entry.isAllFreeVersions() {
				return true
			}
			entries.o = append(entries.o, entry)
			return entries.len() < o.Limit
		})
		if (err != nil && errors.Is(err, io.EOF)) || pastPrefix || r.nextEOF() {
			return entries, io.EOF
		}
		return entries, err
	}

	// We should not need to filter more.
	return r.readN(o.Limit, o.InclDeleted, o.IncludeDirectories, o.Versioned, o.Prefix)
}

func (er *erasureObjects) streamMetadataParts(ctx context.Context, o listPathOptions) (entries metaCacheEntriesSorted, err error) {
	retries := 0
	rpc := globalNotificationSys.restClientFromHash(pathJoin(o.Bucket, o.Prefix))

	const (
		retryDelay    = 50 * time.Millisecond
		retryDelay250 = 250 * time.Millisecond
	)

	for {
		if contextCanceled(ctx) {
			return entries, ctx.Err()
		}

		// If many failures, check the cache state.
		if retries > 10 {
			err := o.checkMetacacheState(ctx, rpc)
			if err != nil {
				return entries, fmt.Errorf("remote listing canceled: %w", err)
			}
			retries = 1
		}

		// All operations are performed without locks, so we must be careful and allow for failures.
		// Read metadata associated with the object from a disk.
		if retries > 0 {
			for _, disk := range er.getDisks() {
				if disk == nil {
					continue
				}
				if !disk.IsOnline() {
					continue
				}
				_, err := disk.ReadVersion(ctx, minioMetaBucket,
					o.objectPath(0), "", false)
				if err != nil {
					time.Sleep(retryDelay250)
					retries++
					continue
				}
				break
			}
		}

		// Load first part metadata...
		// Read metadata associated with the object from all disks.
		fi, metaArr, onlineDisks, err := er.getObjectFileInfo(ctx, minioMetaBucket, o.objectPath(0), ObjectOptions{}, true)
		if err != nil {
			switch toObjectErr(err, minioMetaBucket, o.objectPath(0)).(type) {
			case ObjectNotFound:
				retries++
				if retries == 1 {
					time.Sleep(retryDelay)
				} else {
					time.Sleep(retryDelay250)
				}
				continue
			case InsufficientReadQuorum:
				retries++
				if retries == 1 {
					time.Sleep(retryDelay)
				} else {
					time.Sleep(retryDelay250)
				}
				continue
			default:
				return entries, fmt.Errorf("reading first part metadata: %w", err)
			}
		}

		partN, err := o.findFirstPart(fi)
		switch {
		case err == nil:
		case errors.Is(err, io.ErrUnexpectedEOF):
			if retries == 10 {
				err := o.checkMetacacheState(ctx, rpc)
				if err != nil {
					return entries, fmt.Errorf("remote listing canceled: %w", err)
				}
				retries = -1
			}
			retries++
			time.Sleep(retryDelay250)
			continue
		case errors.Is(err, io.EOF):
			return entries, io.EOF
		}

		// We got a stream to start at.
		loadedPart := 0
		for {
			if contextCanceled(ctx) {
				return entries, ctx.Err()
			}

			if partN != loadedPart {
				if retries > 10 {
					err := o.checkMetacacheState(ctx, rpc)
					if err != nil {
						return entries, fmt.Errorf("waiting for next part %d: %w", partN, err)
					}
					retries = 1
				}

				if retries > 0 {
					// Load from one disk only
					for _, disk := range er.getDisks() {
						if disk == nil {
							continue
						}
						if !disk.IsOnline() {
							continue
						}
						_, err := disk.ReadVersion(ctx, minioMetaBucket,
							o.objectPath(partN), "", false)
						if err != nil {
							time.Sleep(retryDelay250)
							retries++
							continue
						}
						break
					}
				}

				// Load partN metadata...
				fi, metaArr, onlineDisks, err = er.getObjectFileInfo(ctx, minioMetaBucket, o.objectPath(partN), ObjectOptions{}, true)
				if err != nil {
					time.Sleep(retryDelay250)
					retries++
					continue
				}
				loadedPart = partN
				bi, err := getMetacacheBlockInfo(fi, partN)
				logger.LogIf(ctx, err)
				if err == nil {
					if bi.pastPrefix(o.Prefix) {
						return entries, io.EOF
					}
				}
			}

			pr, pw := io.Pipe()
			go func() {
				werr := er.getObjectWithFileInfo(ctx, minioMetaBucket, o.objectPath(partN), 0,
					fi.Size, pw, fi, metaArr, onlineDisks)
				pw.CloseWithError(werr)
			}()

			tmp := newMetacacheReader(pr)
			e, err := tmp.filter(o)
			pr.CloseWithError(err)
			tmp.Close()
			entries.o = append(entries.o, e.o...)
			if o.Limit > 0 && entries.len() > o.Limit {
				entries.truncate(o.Limit)
				return entries, nil
			}
			if err == nil {
				// We stopped within the listing, we are done for now...
				return entries, nil
			}
			if err != nil && !errors.Is(err, io.EOF) {
				switch toObjectErr(err, minioMetaBucket, o.objectPath(partN)).(type) {
				case ObjectNotFound:
					retries++
					time.Sleep(retryDelay250)
					continue
				case InsufficientReadQuorum:
					retries++
					time.Sleep(retryDelay250)
					continue
				default:
					logger.LogIf(ctx, err)
					return entries, err
				}
			}

			// We finished at the end of the block.
			// And should not expect any more results.
			bi, err := getMetacacheBlockInfo(fi, partN)
			logger.LogIf(ctx, err)
			if err != nil || bi.EOS {
				// We are done and there are no more parts.
				return entries, io.EOF
			}
			if bi.endedPrefix(o.Prefix) {
				// Nothing more for prefix.
				return entries, io.EOF
			}
			partN++
			retries = 0
		}
	}
}

// getListQuorum interprets list quorum values and returns appropriate
// acceptable quorum expected for list operations
func getListQuorum(quorum string, driveCount int) int {
	switch quorum {
	case "disk":
		// smallest possible value, generally meant for testing.
		return 1
	case "reduced":
		return 2
	case "strict":
		return driveCount
	}
	// Defaults to (driveCount+1)/2 drives per set, defaults to "optimal" value
	if driveCount > 0 {
		return (driveCount + 1) / 2
	} // "3" otherwise.
	return 3
}

// Will return io.EOF if continuing would not yield more results.
func (er *erasureObjects) listPath(ctx context.Context, o listPathOptions, results chan<- metaCacheEntry) (err error) {
	defer close(results)
	o.debugf(color.Green("listPath:")+" with options: %#v", o)

	// get non-healing disks for listing
	disks, _ := er.getOnlineDisksWithHealing()
	askDisks := getListQuorum(o.AskDisks, er.setDriveCount)
	var fallbackDisks []StorageAPI

	// Special case: ask all disks if the drive count is 4
	if er.setDriveCount == 4 || askDisks > len(disks) {
		askDisks = len(disks) // use all available drives
	}

	// However many we ask, versions must exist on ~50%
	listingQuorum := (askDisks + 1) / 2

	if askDisks > 0 && len(disks) > askDisks {
		rand.Shuffle(len(disks), func(i, j int) {
			disks[i], disks[j] = disks[j], disks[i]
		})
		fallbackDisks = disks[askDisks:]
		disks = disks[:askDisks]
	}

	// How to resolve results.
	resolver := metadataResolutionParams{
		dirQuorum: listingQuorum,
		objQuorum: listingQuorum,
		bucket:    o.Bucket,
	}

	// Maximum versions requested for "latest" object
	// resolution on versioned buckets, this is to be only
	// used when o.Versioned is false
	if !o.Versioned {
		resolver.requestedVersions = 1
	}
	var limit int
	if o.Limit > 0 && o.StopDiskAtLimit {
		// Over-read by 4 + 1 for every 16 in limit to give some space for resolver,
		// allow for truncating the list and know if we have more results.
		limit = o.Limit + 4 + (o.Limit / 16)
	}
	ctxDone := ctx.Done()
	return listPathRaw(ctx, listPathRawOptions{
		disks:         disks,
		fallbackDisks: fallbackDisks,
		bucket:        o.Bucket,
		path:          o.BaseDir,
		recursive:     o.Recursive,
		filterPrefix:  o.FilterPrefix,
		minDisks:      listingQuorum,
		forwardTo:     o.Marker,
		perDiskLimit:  limit,
		agreed: func(entry metaCacheEntry) {
			select {
			case <-ctxDone:
			case results <- entry:
			}
		},
		partial: func(entries metaCacheEntries, errs []error) {
			// Results Disagree :-(
			entry, ok := entries.resolve(&resolver)
			if ok {
				select {
				case <-ctxDone:
				case results <- *entry:
				}
			}
		},
	})
}

type metaCacheRPC struct {
	meta   *metacache
	rpc    *peerRESTClient
	cancel context.CancelFunc
	o      listPathOptions
	mu     sync.Mutex
}

func (m *metaCacheRPC) setErr(err string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	meta := *m.meta
	if meta.status != scanStateError {
		meta.error = err
		meta.status = scanStateError
	} else {
		// An error is already set.
		return
	}
	meta, _ = m.o.updateMetacacheListing(meta, m.rpc)
	*m.meta = meta
}

func (er *erasureObjects) saveMetaCacheStream(ctx context.Context, mc *metaCacheRPC, entries <-chan metaCacheEntry) (err error) {
	o := mc.o
	o.debugf(color.Green("saveMetaCacheStream:")+" with options: %#v", o)

	metaMu := &mc.mu
	rpc := mc.rpc
	cancel := mc.cancel
	defer func() {
		o.debugln(color.Green("saveMetaCacheStream:")+"err:", err)
		if err != nil && !errors.Is(err, io.EOF) {
			go mc.setErr(err.Error())
			cancel()
		}
	}()

	defer cancel()
	// Save continuous updates
	go func() {
		var err error
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		var exit bool
		for !exit {
			select {
			case <-ticker.C:
			case <-ctx.Done():
				exit = true
			}
			metaMu.Lock()
			meta := *mc.meta
			meta, err = o.updateMetacacheListing(meta, rpc)
			if err == nil && time.Since(meta.lastHandout) > metacacheMaxClientWait {
				cancel()
				exit = true
				meta.status = scanStateError
				meta.error = fmt.Sprintf("listing canceled since time since last handout was %v ago", time.Since(meta.lastHandout).Round(time.Second))
				o.debugln(color.Green("saveMetaCacheStream: ") + meta.error)
				meta, err = o.updateMetacacheListing(meta, rpc)
			}
			if err == nil {
				*mc.meta = meta
				if meta.status == scanStateError {
					cancel()
					exit = true
				}
			}
			metaMu.Unlock()
		}
	}()

	const retryDelay = 200 * time.Millisecond
	const maxTries = 5

	// Keep destination...
	// Write results to disk.
	bw := newMetacacheBlockWriter(entries, func(b *metacacheBlock) error {
		// if the block is 0 bytes and its a first block skip it.
		// skip only this for Transient caches.
		if len(b.data) == 0 && b.n == 0 && o.Transient {
			return nil
		}
		o.debugln(color.Green("saveMetaCacheStream:")+" saving block", b.n, "to", o.objectPath(b.n))
		r, err := hash.NewReader(bytes.NewReader(b.data), int64(len(b.data)), "", "", int64(len(b.data)))
		logger.LogIf(ctx, err)
		custom := b.headerKV()
		_, err = er.putMetacacheObject(ctx, o.objectPath(b.n), NewPutObjReader(r), ObjectOptions{
			UserDefined: custom,
		})
		if err != nil {
			mc.setErr(err.Error())
			cancel()
			return err
		}
		if b.n == 0 {
			return nil
		}
		// Update block 0 metadata.
		var retries int
		for {
			meta := b.headerKV()
			fi := FileInfo{
				Metadata: make(map[string]string, len(meta)),
			}
			for k, v := range meta {
				fi.Metadata[k] = v
			}
			err := er.updateObjectMeta(ctx, minioMetaBucket, o.objectPath(0), fi, er.getDisks())
			if err == nil {
				break
			}
			switch err.(type) {
			case ObjectNotFound:
				return err
			case StorageErr:
				return err
			case InsufficientReadQuorum:
			default:
				logger.LogIf(ctx, err)
			}
			if retries >= maxTries {
				return err
			}
			retries++
			time.Sleep(retryDelay)
		}
		return nil
	})

	// Blocks while consuming entries or an error occurs.
	err = bw.Close()
	if err != nil {
		mc.setErr(err.Error())
	}
	metaMu.Lock()
	defer metaMu.Unlock()
	if mc.meta.error != "" {
		return err
	}
	// Save success
	mc.meta.status = scanStateSuccess
	meta, err := o.updateMetacacheListing(*mc.meta, rpc)
	if err == nil {
		*mc.meta = meta
	}
	return nil
}

type listPathRawOptions struct {

	// Callbacks with results:
	// If set to nil, it will not be called.

	// agreed is called if all disks agreed.
	agreed func(entry metaCacheEntry)

	// finished will be called when all streams have finished and
	// more than one disk returned an error.
	// Will not be called if everything operates as expected.
	finished func(errs []error)

	// partial will be called when there is disagreement between disks.
	// if disk did not return any result, but also haven't errored
	// the entry will be empty and errs will
	partial func(entries metaCacheEntries, errs []error)

	// Forward to this prefix before returning results.
	forwardTo string

	// Only return results with this prefix.
	filterPrefix string

	bucket, path  string
	disks         []StorageAPI
	fallbackDisks []StorageAPI

	// Minimum number of good disks to continue.
	// An error will be returned if this many disks returned an error.
	minDisks int

	// perDiskLimit will limit each disk to return n objects.
	// If <= 0 all results will be returned until canceled.
	perDiskLimit int

	recursive bool

	reportNotFound bool
}

// listPathRaw will list a path on the provided drives.
// See listPathRawOptions on how results are delivered.
// Directories are always returned.
// Cache will be bypassed.
// Context cancellation will be respected but may take a while to effectuate.
func listPathRaw(ctx context.Context, opts listPathRawOptions) (err error) {
	disks := opts.disks
	if len(disks) == 0 {
		return fmt.Errorf("listPathRaw: 0 drives provided")
	}

	// Cancel upstream if we finish before we expect.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Keep track of fallback disks
	var fdMu sync.Mutex
	fds := opts.fallbackDisks
	fallback := func(err error) StorageAPI {
		if _, ok := err.(StorageErr); ok {
			// Attempt to grab a fallback disk
			fdMu.Lock()
			defer fdMu.Unlock()
			if len(fds) == 0 {
				return nil
			}
			fdsCopy := fds
			for _, fd := range fdsCopy {
				// Grab a fallback disk
				fds = fds[1:]
				if fd != nil && fd.IsOnline() {
					return fd
				}
			}
		}
		// Either no more disks for fallback or
		// not a storage error.
		return nil
	}
	askDisks := len(disks)
	readers := make([]*metacacheReader, askDisks)
	defer func() {
		for _, r := range readers {
			r.Close()
		}
	}()
	for i := range disks {
		r, w := io.Pipe()
		// Make sure we close the pipe so blocked writes doesn't stay around.
		defer r.CloseWithError(context.Canceled)

		readers[i] = newMetacacheReader(r)
		d := disks[i]

		// Send request to each disk.
		go func() {
			var werr error
			if d == nil {
				werr = errDiskNotFound
			} else {
				werr = d.WalkDir(ctx, WalkDirOptions{
					Limit:          opts.perDiskLimit,
					Bucket:         opts.bucket,
					BaseDir:        opts.path,
					Recursive:      opts.recursive,
					ReportNotFound: opts.reportNotFound,
					FilterPrefix:   opts.filterPrefix,
					ForwardTo:      opts.forwardTo,
				}, w)
			}

			// fallback only when set.
			for {
				fd := fallback(werr)
				if fd == nil {
					break
				}
				// This fallback is only set when
				// askDisks is less than total
				// number of disks per set.
				werr = fd.WalkDir(ctx, WalkDirOptions{
					Limit:          opts.perDiskLimit,
					Bucket:         opts.bucket,
					BaseDir:        opts.path,
					Recursive:      opts.recursive,
					ReportNotFound: opts.reportNotFound,
					FilterPrefix:   opts.filterPrefix,
					ForwardTo:      opts.forwardTo,
				}, w)
				if werr == nil {
					break
				}
			}
			w.CloseWithError(werr)
		}()
	}

	topEntries := make(metaCacheEntries, len(readers))
	errs := make([]error, len(readers))
	for {
		// Get the top entry from each
		var current metaCacheEntry
		var atEOF, fnf, hasErr, agree int
		for i := range topEntries {
			topEntries[i] = metaCacheEntry{}
		}
		if contextCanceled(ctx) {
			return ctx.Err()
		}
		for i, r := range readers {
			if errs[i] != nil {
				hasErr++
				continue
			}
			entry, err := r.peek()
			switch err {
			case io.EOF:
				atEOF++
				continue
			case nil:
			default:
				switch err.Error() {
				case errFileNotFound.Error(),
					errVolumeNotFound.Error(),
					errUnformattedDisk.Error(),
					errDiskNotFound.Error():
					atEOF++
					fnf++
					continue
				}
				hasErr++
				errs[i] = err
				continue
			}
			// If no current, add it.
			if current.name == "" {
				topEntries[i] = entry
				current = entry
				agree++
				continue
			}
			// If exact match, we agree.
			if _, ok := current.matches(&entry, true); ok {
				topEntries[i] = entry
				agree++
				continue
			}
			// If only the name matches we didn't agree, but add it for resolution.
			if entry.name == current.name {
				topEntries[i] = entry
				continue
			}
			// We got different entries
			if entry.name > current.name {
				continue
			}
			// We got a new, better current.
			// Clear existing entries.
			for i := range topEntries[:i] {
				topEntries[i] = metaCacheEntry{}
			}
			agree = 1
			current = entry
			topEntries[i] = entry
		}

		// Stop if we exceed number of bad disks
		if hasErr > len(disks)-opts.minDisks && hasErr > 0 {
			if opts.finished != nil {
				opts.finished(errs)
			}
			var combinedErr []string
			for i, err := range errs {
				if err != nil {
					if disks[i] != nil {
						combinedErr = append(combinedErr,
							fmt.Sprintf("drive %s returned: %s", disks[i], err))
					} else {
						combinedErr = append(combinedErr, err.Error())
					}
				}
			}
			return errors.New(strings.Join(combinedErr, ", "))
		}

		// Break if all at EOF or error.
		if atEOF+hasErr == len(readers) {
			if hasErr > 0 && opts.finished != nil {
				opts.finished(errs)
			}
			break
		}
		if fnf == len(readers) {
			return errFileNotFound
		}
		if agree == len(readers) {
			// Everybody agreed
			for _, r := range readers {
				r.skip(1)
			}
			if opts.agreed != nil {
				opts.agreed(current)
			}
			continue
		}
		if opts.partial != nil {
			opts.partial(topEntries, errs)
		}
		// Skip the inputs we used.
		for i, r := range readers {
			if topEntries[i].name != "" {
				r.skip(1)
			}
		}
	}
	return nil
}
