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
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/google/uuid"
	"github.com/minio/minio/internal/bucket/lifecycle"
	"github.com/minio/minio/internal/bucket/replication"
	xhttp "github.com/minio/minio/internal/http"
	"github.com/minio/minio/internal/logger"
	"github.com/tinylib/msgp/msgp"
)

var (
	// XL header specifies the format
	xlHeader = [4]byte{'X', 'L', '2', ' '}

	// Current version being written.
	xlVersionCurrent [4]byte
)

const (
	// Breaking changes.
	// Newer versions cannot be read by older software.
	// This will prevent downgrades to incompatible versions.
	xlVersionMajor = 1

	// Non breaking changes.
	// Bumping this is informational, but should be done
	// if any change is made to the data stored, bumping this
	// will allow to detect the exact version later.
	xlVersionMinor = 3
)

func init() {
	binary.LittleEndian.PutUint16(xlVersionCurrent[0:2], xlVersionMajor)
	binary.LittleEndian.PutUint16(xlVersionCurrent[2:4], xlVersionMinor)
}

// checkXL2V1 will check if the metadata has correct header and is a known major version.
// The remaining payload and versions are returned.
func checkXL2V1(buf []byte) (payload []byte, major, minor uint16, err error) {
	if len(buf) <= 8 {
		return payload, 0, 0, fmt.Errorf("xlMeta: no data")
	}

	if !bytes.Equal(buf[:4], xlHeader[:]) {
		return payload, 0, 0, fmt.Errorf("xlMeta: unknown XLv2 header, expected %v, got %v", xlHeader[:4], buf[:4])
	}

	if bytes.Equal(buf[4:8], []byte("1   ")) {
		// Set as 1,0.
		major, minor = 1, 0
	} else {
		major, minor = binary.LittleEndian.Uint16(buf[4:6]), binary.LittleEndian.Uint16(buf[6:8])
	}
	if major > xlVersionMajor {
		return buf[8:], major, minor, fmt.Errorf("xlMeta: unknown major version %d found", major)
	}

	return buf[8:], major, minor, nil
}

func isXL2V1Format(buf []byte) bool {
	_, _, _, err := checkXL2V1(buf)
	return err == nil
}

// The []journal contains all the different versions of the object.
//
// This array can have 3 kinds of objects:
//
// ``object``: If the object is uploaded the usual way: putobject, multipart-put, copyobject
//
// ``delete``: This is the delete-marker
//
// ``legacyObject``: This is the legacy object in xlV1 format, preserved until its overwritten
//
// The most recently updated element in the array is considered the latest version.

// In addition to these we have a special kind called free-version. This is represented
// using a delete-marker and MetaSys entries. It's used to track tiered content of a
// deleted/overwritten version. This version is visible _only_to the scanner routine, for subsequent deletion.
// This kind of tracking is necessary since a version's tiered content is deleted asynchronously.

// Backend directory tree structure:
// disk1/
// └── bucket
//     └── object
//         ├── a192c1d5-9bd5-41fd-9a90-ab10e165398d
//         │   └── part.1
//         ├── c06e0436-f813-447e-ae5e-f2564df9dfd4
//         │   └── part.1
//         ├── df433928-2dcf-47b1-a786-43efa0f6b424
//         │   └── part.1
//         ├── legacy
//         │   └── part.1
//         └── xl.meta

//go:generate msgp -file=$GOFILE -unexported

// VersionType defines the type of journal type of the current entry.
type VersionType uint8

// List of different types of journal type
const (
	invalidVersionType VersionType = 0
	ObjectType         VersionType = 1
	DeleteType         VersionType = 2
	LegacyType         VersionType = 3
	lastVersionType    VersionType = 4
)

func (e VersionType) valid() bool {
	return e > invalidVersionType && e < lastVersionType
}

// ErasureAlgo defines common type of different erasure algorithms
type ErasureAlgo uint8

// List of currently supported erasure coding algorithms
const (
	invalidErasureAlgo ErasureAlgo = 0
	ReedSolomon        ErasureAlgo = 1
	lastErasureAlgo    ErasureAlgo = 2
)

func (e ErasureAlgo) valid() bool {
	return e > invalidErasureAlgo && e < lastErasureAlgo
}

func (e ErasureAlgo) String() string {
	switch e {
	case ReedSolomon:
		return "reedsolomon"
	}
	return ""
}

// ChecksumAlgo defines common type of different checksum algorithms
type ChecksumAlgo uint8

// List of currently supported checksum algorithms
const (
	invalidChecksumAlgo ChecksumAlgo = 0
	HighwayHash         ChecksumAlgo = 1
	lastChecksumAlgo    ChecksumAlgo = 2
)

func (e ChecksumAlgo) valid() bool {
	return e > invalidChecksumAlgo && e < lastChecksumAlgo
}

// xlMetaV2DeleteMarker defines the data struct for the delete marker journal type
type xlMetaV2DeleteMarker struct {
	VersionID [16]byte          `json:"ID" msg:"ID"`                               // Version ID for delete marker
	ModTime   int64             `json:"MTime" msg:"MTime"`                         // Object delete marker modified time
	MetaSys   map[string][]byte `json:"MetaSys,omitempty" msg:"MetaSys,omitempty"` // Delete marker internal metadata
}

// xlMetaV2Object defines the data struct for object journal type
type xlMetaV2Object struct {
	VersionID          [16]byte          `json:"ID" msg:"ID"`                                    // Version ID
	DataDir            [16]byte          `json:"DDir" msg:"DDir"`                                // Data dir ID
	ErasureAlgorithm   ErasureAlgo       `json:"EcAlgo" msg:"EcAlgo"`                            // Erasure coding algorithm
	ErasureM           int               `json:"EcM" msg:"EcM"`                                  // Erasure data blocks
	ErasureN           int               `json:"EcN" msg:"EcN"`                                  // Erasure parity blocks
	ErasureBlockSize   int64             `json:"EcBSize" msg:"EcBSize"`                          // Erasure block size
	ErasureIndex       int               `json:"EcIndex" msg:"EcIndex"`                          // Erasure disk index
	ErasureDist        []uint8           `json:"EcDist" msg:"EcDist"`                            // Erasure distribution
	BitrotChecksumAlgo ChecksumAlgo      `json:"CSumAlgo" msg:"CSumAlgo"`                        // Bitrot checksum algo
	PartNumbers        []int             `json:"PartNums" msg:"PartNums"`                        // Part Numbers
	PartETags          []string          `json:"PartETags" msg:"PartETags"`                      // Part ETags
	PartSizes          []int64           `json:"PartSizes" msg:"PartSizes"`                      // Part Sizes
	PartActualSizes    []int64           `json:"PartASizes,omitempty" msg:"PartASizes,allownil"` // Part ActualSizes (compression)
	Size               int64             `json:"Size" msg:"Size"`                                // Object version size
	ModTime            int64             `json:"MTime" msg:"MTime"`                              // Object version modified time
	MetaSys            map[string][]byte `json:"MetaSys,omitempty" msg:"MetaSys,allownil"`       // Object version internal metadata
	MetaUser           map[string]string `json:"MetaUsr,omitempty" msg:"MetaUsr,allownil"`       // Object version metadata set by user
}

// xlMetaV2Version describes the journal entry, Type defines
// the current journal entry type other types might be nil based
// on what Type field carries, it is imperative for the caller
// to verify which journal type first before accessing rest of the fields.
type xlMetaV2Version struct {
	Type         VersionType           `json:"Type" msg:"Type"`
	ObjectV1     *xlMetaV1Object       `json:"V1Obj,omitempty" msg:"V1Obj,omitempty"`
	ObjectV2     *xlMetaV2Object       `json:"V2Obj,omitempty" msg:"V2Obj,omitempty"`
	DeleteMarker *xlMetaV2DeleteMarker `json:"DelObj,omitempty" msg:"DelObj,omitempty"`
}

// xlFlags contains flags on the object.
// This can be extended up to 64 bits without breaking compatibility.
type xlFlags uint8

const (
	xlFlagFreeVersion xlFlags = 1 << iota
	xlFlagUsesDataDir
)

//msgp:tuple xlMetaV2VersionHeader
type xlMetaV2VersionHeader struct {
	VersionID [16]byte
	ModTime   int64
	Type      VersionType
	Flags     xlFlags
}

// Valid xl meta xlMetaV2Version is valid
func (j xlMetaV2Version) Valid() bool {
	if !j.Type.valid() {
		return false
	}
	switch j.Type {
	case LegacyType:
		return j.ObjectV1 != nil &&
			j.ObjectV1.valid()
	case ObjectType:
		return j.ObjectV2 != nil &&
			j.ObjectV2.ErasureAlgorithm.valid() &&
			j.ObjectV2.BitrotChecksumAlgo.valid() &&
			isXLMetaErasureInfoValid(j.ObjectV2.ErasureM, j.ObjectV2.ErasureN) &&
			j.ObjectV2.ModTime > 0
	case DeleteType:
		return j.DeleteMarker != nil &&
			j.DeleteMarker.ModTime > 0
	}
	return false
}

// header will return a shallow header of the version.
func (j *xlMetaV2Version) header() xlMetaV2VersionHeader {
	var flags xlFlags
	if j.FreeVersion() {
		flags |= xlFlagFreeVersion
	}
	if j.Type == ObjectType && j.ObjectV2.UsesDataDir() {
		flags |= xlFlagUsesDataDir
	}
	return xlMetaV2VersionHeader{
		Type:      j.Type,
		ModTime:   j.getModTime().UnixNano(),
		VersionID: j.getVersionID(),
		Flags:     flags,
	}
}

func (x xlMetaV2VersionHeader) FreeVersion() bool {
	return x.Flags&xlFlagFreeVersion != 0
}

func (x xlMetaV2VersionHeader) UsesDataDir() bool {
	return x.Flags&xlFlagUsesDataDir != 0
}

// getModTime will return the ModTime of the underlying version.
func (j xlMetaV2Version) getModTime() time.Time {
	switch j.Type {
	case ObjectType:
		return time.Unix(0, j.ObjectV2.ModTime)
	case DeleteType:
		return time.Unix(0, j.DeleteMarker.ModTime)
	case LegacyType:
		return j.ObjectV1.Stat.ModTime
	}
	return time.Time{}
}

// getModTime will return the ModTime of the underlying version.
func (j xlMetaV2Version) getVersionID() [16]byte {
	switch j.Type {
	case ObjectType:
		return j.ObjectV2.VersionID
	case DeleteType:
		return j.DeleteMarker.VersionID
	case LegacyType:
		return [16]byte{}
	}
	return [16]byte{}
}

func (j xlMetaV2Version) ToFileInfo(volume, path string) (FileInfo, error) {
	switch j.Type {
	case ObjectType:
		return j.ObjectV2.ToFileInfo(volume, path)
	case DeleteType:
		return j.DeleteMarker.ToFileInfo(volume, path)
	case LegacyType:
		return j.ObjectV1.ToFileInfo(volume, path)
	}
	return FileInfo{}, errFileNotFound
}

// xlMetaV2 - object meta structure defines the format and list of
// the journals for the object.
type xlMetaV2 struct {
	Versions []xlMetaV2Version `json:"Versions" msg:"Versions"`

	// data will contain raw data if any.
	// data will be one or more versions indexed by versionID.
	// To remove all data set to nil.
	data xlMetaInlineData `msg:"-"`
}

// Load unmarshal and load the entire message pack.
// Note that references to the incoming buffer may be kept as data.
func (z *xlMetaV2) Load(buf []byte) error {
	buf, major, minor, err := checkXL2V1(buf)
	if err != nil {
		return fmt.Errorf("xlMetaV2.Load %w", err)
	}
	switch major {
	case 1:
		switch minor {
		case 0:
			_, err = z.UnmarshalMsg(buf)
			if err != nil {
				return fmt.Errorf("xlMetaV2.Load %w", err)
			}
			return nil
		case 1, 2, 3:
			v, buf, err := msgp.ReadBytesZC(buf)
			if err != nil {
				return fmt.Errorf("xlMetaV2.Load version(%d), bufLen(%d) %w", minor, len(buf), err)
			}
			if minor >= 2 {
				if crc, nbuf, err := msgp.ReadUint32Bytes(buf); err == nil {
					// Read metadata CRC (added in v2)
					buf = nbuf
					if got := uint32(xxhash.Sum64(v)); got != crc {
						return fmt.Errorf("xlMetaV2.Load version(%d), CRC mismatch, want 0x%x, got 0x%x", minor, crc, got)
					}
				} else {
					return fmt.Errorf("xlMetaV2.Load version(%d), loading CRC: %w", minor, err)
				}
			}

			if minor < 3 {
				if _, err = z.UnmarshalMsg(v); err != nil {
					return fmt.Errorf("xlMetaV2.Load version(%d), vLen(%d), %w", minor, len(v), err)
				}
				z.sortByModtime()
			} else {
				if err = z.loadWithIndex(v); err != nil {
					return fmt.Errorf("xlMetaV2.Load version(%d), vLen(%d), err: %w", minor, len(v), err)
				}
			}
			// Add remaining data.
			z.data = buf
			if err = z.data.validate(); err != nil {
				z.data.repair()
				logger.Info("xlMetaV2.Load: data validation failed: %v. %d entries after repair", err, z.data.entries())
			}
		default:
			return errors.New("unknown minor metadata version")
		}
	default:
		return errors.New("unknown major metadata version")
	}
	return nil
}

const (
	xlHeaderVersion = 1
	xlMetaVersion   = 1
)

func (z *xlMetaV2) loadWithIndex(buf []byte) error {
	versions, buf, err := decodeXlHeaders(buf)
	if err != nil {
		return err
	}
	if cap(z.Versions) < versions {
		z.Versions = make([]xlMetaV2Version, 0, versions)
	}
	z.Versions = z.Versions[:versions]
	return decodeVersions(buf, versions, func(idx int, hdr, meta []byte) error {
		// Unmarshal directly.
		ver := &z.Versions[idx]
		*ver = xlMetaV2Version{}
		_, err = ver.UnmarshalMsg(meta)
		if err != nil {
			return err
		}
		return nil
	})
}

func (z *xlMetaV2) asShallow() (*xlMetaV2Shallow, error) {
	res := xlMetaV2Shallow{
		versions: make([]xlmetaV2ShallowVersion, 0, len(z.Versions)),
		data:     z.data,
	}
	for _, ver := range z.Versions {
		if !ver.Valid() {
			return nil, errFileCorrupt
		}
		meta, err := ver.MarshalMsg(nil)
		if err != nil {
			return nil, err
		}

		res.versions = append(res.versions, xlmetaV2ShallowVersion{
			header: ver.header(),
			meta:   meta,
		})
	}
	return &res, nil
}

func (j xlMetaV2DeleteMarker) ToFileInfo(volume, path string) (FileInfo, error) {
	versionID := ""
	var uv uuid.UUID
	// check if the version is not "null"
	if j.VersionID != uv {
		versionID = uuid.UUID(j.VersionID).String()
	}
	fi := FileInfo{
		Volume:    volume,
		Name:      path,
		ModTime:   time.Unix(0, j.ModTime).UTC(),
		VersionID: versionID,
		Deleted:   true,
	}
	fi.ReplicationState = GetInternalReplicationState(j.MetaSys)

	if j.FreeVersion() {
		fi.SetTierFreeVersion()
		fi.TransitionTier = string(j.MetaSys[ReservedMetadataPrefixLower+TransitionTier])
		fi.TransitionedObjName = string(j.MetaSys[ReservedMetadataPrefixLower+TransitionedObjectName])
		fi.TransitionVersionID = string(j.MetaSys[ReservedMetadataPrefixLower+TransitionedVersionID])
	}

	return fi, nil
}

// UsesDataDir returns true if this object version uses its data directory for
// its contents and false otherwise.
func (j xlMetaV2Object) UsesDataDir() bool {
	// Skip if this version is not transitioned, i.e it uses its data directory.
	if !bytes.Equal(j.MetaSys[ReservedMetadataPrefixLower+TransitionStatus], []byte(lifecycle.TransitionComplete)) {
		return true
	}

	// Check if this transitioned object has been restored on disk.
	return isRestoredObjectOnDisk(j.MetaUser)
}

func (j *xlMetaV2Object) SetTransition(fi FileInfo) {
	j.MetaSys[ReservedMetadataPrefixLower+TransitionStatus] = []byte(fi.TransitionStatus)
	j.MetaSys[ReservedMetadataPrefixLower+TransitionedObjectName] = []byte(fi.TransitionedObjName)
	j.MetaSys[ReservedMetadataPrefixLower+TransitionedVersionID] = []byte(fi.TransitionVersionID)
	j.MetaSys[ReservedMetadataPrefixLower+TransitionTier] = []byte(fi.TransitionTier)
}

func (j *xlMetaV2Object) RemoveRestoreHdrs() {
	delete(j.MetaUser, xhttp.AmzRestore)
	delete(j.MetaUser, xhttp.AmzRestoreExpiryDays)
	delete(j.MetaUser, xhttp.AmzRestoreRequestDate)
}

func (j xlMetaV2Object) ToFileInfo(volume, path string) (FileInfo, error) {
	versionID := ""
	var uv uuid.UUID
	// check if the version is not "null"
	if j.VersionID != uv {
		versionID = uuid.UUID(j.VersionID).String()
	}
	fi := FileInfo{
		Volume:    volume,
		Name:      path,
		Size:      j.Size,
		ModTime:   time.Unix(0, j.ModTime).UTC(),
		VersionID: versionID,
	}
	fi.Parts = make([]ObjectPartInfo, len(j.PartNumbers))
	for i := range fi.Parts {
		fi.Parts[i].Number = j.PartNumbers[i]
		fi.Parts[i].Size = j.PartSizes[i]
		fi.Parts[i].ETag = j.PartETags[i]
		fi.Parts[i].ActualSize = j.PartActualSizes[i]
	}
	fi.Erasure.Checksums = make([]ChecksumInfo, len(j.PartSizes))
	for i := range fi.Parts {
		fi.Erasure.Checksums[i].PartNumber = fi.Parts[i].Number
		switch j.BitrotChecksumAlgo {
		case HighwayHash:
			fi.Erasure.Checksums[i].Algorithm = HighwayHash256S
			fi.Erasure.Checksums[i].Hash = []byte{}
		default:
			return FileInfo{}, fmt.Errorf("unknown BitrotChecksumAlgo: %v", j.BitrotChecksumAlgo)
		}
	}
	fi.Metadata = make(map[string]string, len(j.MetaUser)+len(j.MetaSys))
	for k, v := range j.MetaUser {
		// https://github.com/google/security-research/security/advisories/GHSA-76wf-9vgp-pj7w
		if equals(k, xhttp.AmzMetaUnencryptedContentLength, xhttp.AmzMetaUnencryptedContentMD5) {
			continue
		}

		fi.Metadata[k] = v
	}
	for k, v := range j.MetaSys {
		switch {
		case strings.HasPrefix(strings.ToLower(k), ReservedMetadataPrefixLower), equals(k, VersionPurgeStatusKey):
			fi.Metadata[k] = string(v)
		}
	}
	fi.ReplicationState = getInternalReplicationState(fi.Metadata)
	replStatus := fi.ReplicationState.CompositeReplicationStatus()
	if replStatus != "" {
		fi.Metadata[xhttp.AmzBucketReplicationStatus] = string(replStatus)
	}
	fi.Erasure.Algorithm = j.ErasureAlgorithm.String()
	fi.Erasure.Index = j.ErasureIndex
	fi.Erasure.BlockSize = j.ErasureBlockSize
	fi.Erasure.DataBlocks = j.ErasureM
	fi.Erasure.ParityBlocks = j.ErasureN
	fi.Erasure.Distribution = make([]int, len(j.ErasureDist))
	for i := range j.ErasureDist {
		fi.Erasure.Distribution[i] = int(j.ErasureDist[i])
	}
	fi.DataDir = uuid.UUID(j.DataDir).String()

	if st, ok := j.MetaSys[ReservedMetadataPrefixLower+TransitionStatus]; ok {
		fi.TransitionStatus = string(st)
	}
	if o, ok := j.MetaSys[ReservedMetadataPrefixLower+TransitionedObjectName]; ok {
		fi.TransitionedObjName = string(o)
	}
	if rv, ok := j.MetaSys[ReservedMetadataPrefixLower+TransitionedVersionID]; ok {
		fi.TransitionVersionID = string(rv)
	}
	if sc, ok := j.MetaSys[ReservedMetadataPrefixLower+TransitionTier]; ok {
		fi.TransitionTier = string(sc)
	}
	return fi, nil
}

// sortByModtime will sort versions by modtime in descending order,
// meaning index 0 will be latest version.
func (z *xlMetaV2) sortByModtime() {
	// Quick check
	if sort.SliceIsSorted(z.Versions, func(i, j int) bool {
		return z.Versions[i].getModTime().After(z.Versions[j].getModTime())
	}) {
		return
	}

	// We should.
	sort.Slice(z.Versions, func(i, j int) bool {
		return z.Versions[i].getModTime().After(z.Versions[j].getModTime())
	})
}

// Read at most this much on initial read.
const metaDataReadDefault = 4 << 10

// Return used metadata byte slices here.
var metaDataPool = sync.Pool{New: func() interface{} { return make([]byte, 0, metaDataReadDefault) }}

// metaDataPoolGet will return a byte slice with capacity at least metaDataReadDefault.
// It will be length 0.
func metaDataPoolGet() []byte {
	return metaDataPool.Get().([]byte)[:0]
}

// metaDataPoolPut will put an unused small buffer back into the pool.
func metaDataPoolPut(buf []byte) {
	if cap(buf) >= metaDataReadDefault && cap(buf) < metaDataReadDefault*4 {
		metaDataPool.Put(buf)
	}
}

// readXLMetaNoData will load the metadata, but skip data segments.
// This should only be used when data is never interesting.
// If data is not xlv2, it is returned in full.
func readXLMetaNoData(r io.Reader, size int64) ([]byte, error) {
	initial := size
	hasFull := true
	if initial > metaDataReadDefault {
		initial = metaDataReadDefault
		hasFull = false
	}

	buf := metaDataPoolGet()[:initial]
	_, err := io.ReadFull(r, buf)
	if err != nil {
		return nil, fmt.Errorf("readXLMetaNoData.ReadFull: %w", err)
	}
	readMore := func(n int64) error {
		has := int64(len(buf))
		if has >= n {
			return nil
		}
		if hasFull || n > size {
			return io.ErrUnexpectedEOF
		}
		extra := n - has
		buf = append(buf, make([]byte, extra)...)
		_, err := io.ReadFull(r, buf[has:])
		if err != nil {
			if errors.Is(err, io.EOF) {
				// Returned if we read nothing.
				return fmt.Errorf("readXLMetaNoData.readMore: %w", io.ErrUnexpectedEOF)
			}
			return fmt.Errorf("readXLMetaNoData.readMore: %w", err)
		}
		return nil
	}
	tmp, major, minor, err := checkXL2V1(buf)
	if err != nil {
		err = readMore(size)
		return buf, err
	}
	switch major {
	case 1:
		switch minor {
		case 0:
			err = readMore(size)
			return buf, err
		case 1, 2, 3:
			sz, tmp, err := msgp.ReadBytesHeader(tmp)
			if err != nil {
				return nil, err
			}
			want := int64(sz) + int64(len(buf)-len(tmp))

			// v1.1 does not have CRC.
			if minor < 2 {
				if err := readMore(want); err != nil {
					return nil, err
				}
				return buf[:want], nil
			}

			// CRC is variable length, so we need to truncate exactly that.
			wantMax := want + msgp.Uint32Size
			if wantMax > size {
				wantMax = size
			}
			if err := readMore(wantMax); err != nil {
				return nil, err
			}

			tmp = buf[want:]
			_, after, err := msgp.ReadUint32Bytes(tmp)
			if err != nil {
				return nil, err
			}
			want += int64(len(tmp) - len(after))

			return buf[:want], err

		default:
			return nil, errors.New("unknown minor metadata version")
		}
	default:
		return nil, errors.New("unknown major metadata version")
	}
}

func decodeXlHeaders(buf []byte) (versions int, b []byte, err error) {
	hdrVer, buf, err := msgp.ReadUintBytes(buf)
	if err != nil {
		return 0, buf, err
	}
	metaVer, buf, err := msgp.ReadUintBytes(buf)
	if err != nil {
		return 0, buf, err
	}
	if hdrVer > xlHeaderVersion {
		return 0, buf, fmt.Errorf("decodeXlHeaders: Unknown xl header version %d", metaVer)
	}
	if metaVer > xlMetaVersion {
		return 0, buf, fmt.Errorf("decodeXlHeaders: Unknown xl meta version %d", metaVer)
	}
	versions, buf, err = msgp.ReadIntBytes(buf)
	if err != nil {
		return 0, buf, err
	}
	if versions < 0 {
		return 0, buf, fmt.Errorf("decodeXlHeaders: Negative version count %d", versions)
	}
	return versions, buf, nil
}

// decodeVersions will decode a number of versions from a buffer
// and perform a callback for each version in order, newest first.
// Return errDoneForNow to stop processing and return nil.
// Any non-nil error is returned.
func decodeVersions(buf []byte, versions int, fn func(idx int, hdr, meta []byte) error) (err error) {
	var tHdr, tMeta []byte // Zero copy bytes
	for i := 0; i < versions; i++ {
		tHdr, buf, err = msgp.ReadBytesZC(buf)
		if err != nil {
			return err
		}
		tMeta, buf, err = msgp.ReadBytesZC(buf)
		if err != nil {
			return err
		}
		if err = fn(i, tHdr, tMeta); err != nil {
			if err == errDoneForNow {
				err = nil
			}
			return err
		}
	}
	return nil
}

// isIndexedMetaV2 returns non-nil result if metadata is indexed.
// If data doesn't validate nil is also returned.
func isIndexedMetaV2(buf []byte) (meta xlMetaBuf, data xlMetaInlineData) {
	buf, major, minor, err := checkXL2V1(buf)
	if err != nil {
		return nil, nil
	}
	if major != 1 && minor < 3 {
		return nil, nil
	}
	meta, buf, err = msgp.ReadBytesZC(buf)
	if err != nil {
		return nil, nil
	}
	if crc, nbuf, err := msgp.ReadUint32Bytes(buf); err == nil {
		// Read metadata CRC
		buf = nbuf
		if got := uint32(xxhash.Sum64(meta)); got != crc {
			return nil, nil
		}
	} else {
		return nil, nil
	}
	data = buf
	if data.validate() != nil {
		data.repair()
	}

	return meta, data
}

type xlmetaV2ShallowVersion struct {
	header xlMetaV2VersionHeader
	meta   []byte
}

//msgp:ignore xlMetaV2Shallow xlmetaV2ShallowVersion

type xlMetaV2Shallow struct {
	versions []xlmetaV2ShallowVersion

	// data will contain raw data if any.
	// data will be one or more versions indexed by versionID.
	// To remove all data set to nil.
	data xlMetaInlineData
}

func (x *xlMetaV2Shallow) Load(buf []byte) error {
	if meta, data := isIndexedMetaV2(buf); meta != nil {
		return x.loadVersions(meta, data)
	}
	// Convert older format.
	var xl xlMetaV2
	if err := xl.Load(buf); err != nil {
		return err
	}
	shallow, err := xl.asShallow()
	if err != nil {
		return err
	}
	*x = *shallow
	return nil
}

func (x *xlMetaV2Shallow) loadVersions(buf xlMetaBuf, data xlMetaInlineData) error {
	versions, buf, err := decodeXlHeaders(buf)
	if err != nil {
		return err
	}
	if cap(x.versions) < versions {
		x.versions = make([]xlmetaV2ShallowVersion, 0, versions)
	}
	x.versions = x.versions[:versions]
	x.data = data
	if err = x.data.validate(); err != nil {
		x.data.repair()
		logger.Info("xlMetaV2Shallow.loadVersions: data validation failed: %v. %d entries after repair", err, x.data.entries())
	}

	return decodeVersions(buf, versions, func(i int, hdr, meta []byte) error {
		ver := &x.versions[i]
		_, err = ver.header.UnmarshalMsg(hdr)
		if err != nil {
			return err
		}
		ver.meta = meta
		return nil
	})
}

func (x *xlMetaV2Shallow) addVersion(ver xlMetaV2Version) error {
	modTime := ver.getModTime().UnixNano()
	if !ver.Valid() {
		return errors.New("attempted to add invalid version")
	}
	encoded, err := ver.MarshalMsg(nil)
	if err != nil {
		return err
	}
	// Add space at the end.
	// Will have -1 modtime, so it will be inserted there.
	x.versions = append(x.versions, xlmetaV2ShallowVersion{header: xlMetaV2VersionHeader{ModTime: -1}})

	// Linear search, we likely have to insert at front.
	for i, existing := range x.versions {
		if existing.header.ModTime <= modTime {
			// Insert at current idx. First move current back.
			copy(x.versions[i+1:], x.versions[i:])
			x.versions[i] = xlmetaV2ShallowVersion{
				header: ver.header(),
				meta:   encoded,
			}
			return nil
		}
	}
	return fmt.Errorf("addVersion: Internal error, unable to add version")
}

// AppendTo will marshal the data in z and append it to the provided slice.
func (x *xlMetaV2Shallow) AppendTo(dst []byte) ([]byte, error) {
	// TODO: Make more precise
	sz := len(xlHeader) + len(xlVersionCurrent) + msgp.ArrayHeaderSize + len(dst) + msgp.Uint32Size
	if cap(dst) < sz {
		buf := make([]byte, len(dst), sz)
		copy(buf, dst)
		dst = buf
	}
	if err := x.data.validate(); err != nil {
		return nil, err
	}

	dst = append(dst, xlHeader[:]...)
	dst = append(dst, xlVersionCurrent[:]...)
	// Add "bin 32" type header to always have enough space.
	// We will fill out the correct size when we know it.
	dst = append(dst, 0xc6, 0, 0, 0, 0)
	dataOffset := len(dst)

	dst = msgp.AppendUint(dst, xlHeaderVersion)
	dst = msgp.AppendUint(dst, xlMetaVersion)
	dst = msgp.AppendInt(dst, len(x.versions))

	tmp := metaDataPoolGet()
	defer metaDataPoolPut(tmp)
	for _, ver := range x.versions {
		var err error

		// Add header
		tmp, err = ver.header.MarshalMsg(tmp[:0])
		if err != nil {
			return nil, err
		}
		dst = msgp.AppendBytes(dst, tmp)

		// Add full meta
		dst = msgp.AppendBytes(dst, ver.meta)
	}

	// Update size...
	binary.BigEndian.PutUint32(dst[dataOffset-4:dataOffset], uint32(len(dst)-dataOffset))

	// Add CRC of metadata.
	dst = msgp.AppendUint32(dst, uint32(xxhash.Sum64(dst[dataOffset:])))
	return append(dst, x.data...), nil
}

func (x *xlMetaV2Shallow) findVersion(key [16]byte) (idx int, ver *xlMetaV2Version, err error) {
	for i, ver := range x.versions {
		if key == ver.header.VersionID {
			obj, err := x.getIdx(i)
			return i, obj, err
		}
	}
	return -1, nil, errFileVersionNotFound
}

func (x *xlMetaV2Shallow) getIdx(idx int) (ver *xlMetaV2Version, err error) {
	if idx < 0 || idx >= len(x.versions) {
		return nil, errFileNotFound
	}
	var dst xlMetaV2Version
	_, err = dst.UnmarshalMsg(x.versions[idx].meta)
	if false {
		if err == nil && x.versions[idx].header.VersionID != dst.getVersionID() {
			panic(fmt.Sprintf("header: %x != object id: %x", x.versions[idx].header.VersionID, dst.getVersionID()))
		}
	}
	return &dst, err
}

func (x *xlMetaV2Shallow) getVersion(versionID [16]byte) (idx int, ver *xlMetaV2Version) {
	for i := range x.versions {
		if x.versions[i].header.VersionID == versionID {
			var dst xlMetaV2Version
			if _, err := dst.UnmarshalMsg(x.versions[i].meta); err != nil {
				return -1, nil
			}
			if true {
				if dst.getVersionID() != versionID {
					panic(fmt.Sprintf("%x != %x", dst.getVersionID(), versionID))
				}
			}
			return i, &dst
		}
	}
	return -1, nil
}

// setIdx will replace a version at a given index.
// Note that versions may become re-sorted if modtime changes.
func (x *xlMetaV2Shallow) setIdx(idx int, ver xlMetaV2Version) (err error) {
	if idx < 0 || idx >= len(x.versions) {
		return errFileNotFound
	}
	update := &x.versions[idx]
	prevMod := update.header.ModTime
	update.meta, err = ver.MarshalMsg(update.meta[:0:len(update.meta)])
	if err != nil {
		update.meta = nil
		return err
	}
	update.header = ver.header()
	if prevMod != update.header.ModTime {
		x.sortByModTime()
	}
	return nil
}

// sortByModTime will sort versions by modtime in descending order,
// meaning index 0 will be latest version.
func (z *xlMetaV2Shallow) sortByModTime() {
	// Quick check
	if len(z.versions) <= 1 || sort.SliceIsSorted(z.versions, func(i, j int) bool {
		return z.versions[i].header.ModTime > z.versions[j].header.ModTime
	}) {
		return
	}

	// We should sort.
	sort.Slice(z.versions, func(i, j int) bool {
		return z.versions[i].header.ModTime > z.versions[j].header.ModTime
	})
}

// DeleteVersion deletes the version specified by version id.
// returns to the caller which dataDir to delete, also
// indicates if this is the last version.
func (x *xlMetaV2Shallow) DeleteVersion(fi FileInfo) (string, bool, error) {
	// This is a situation where versionId is explicitly
	// specified as "null", as we do not save "null"
	// string it is considered empty. But empty also
	// means the version which matches will be purged.
	if fi.VersionID == nullVersionID {
		fi.VersionID = ""
	}

	var uv uuid.UUID
	var err error
	if fi.VersionID != "" {
		uv, err = uuid.Parse(fi.VersionID)
		if err != nil {
			return "", false, errFileVersionNotFound
		}
	}

	var ventry xlMetaV2Version
	if fi.Deleted {
		ventry = xlMetaV2Version{
			Type: DeleteType,
			DeleteMarker: &xlMetaV2DeleteMarker{
				VersionID: uv,
				ModTime:   fi.ModTime.UnixNano(),
				MetaSys:   make(map[string][]byte),
			},
		}
		if !ventry.Valid() {
			return "", false, errors.New("internal error: invalid version entry generated")
		}
	}
	updateVersion := false
	if fi.VersionPurgeStatus().Empty() && (fi.DeleteMarkerReplicationStatus() == "REPLICA" || fi.DeleteMarkerReplicationStatus().Empty()) {
		updateVersion = fi.MarkDeleted
	} else {
		// for replication scenario
		if fi.Deleted && fi.VersionPurgeStatus() != Complete {
			if !fi.VersionPurgeStatus().Empty() || fi.DeleteMarkerReplicationStatus().Empty() {
				updateVersion = true
			}
		}
		// object or delete-marker versioned delete is not complete
		if !fi.VersionPurgeStatus().Empty() && fi.VersionPurgeStatus() != Complete {
			updateVersion = true
		}
	}

	if fi.Deleted {
		if !fi.DeleteMarkerReplicationStatus().Empty() {
			switch fi.DeleteMarkerReplicationStatus() {
			case replication.Replica:
				ventry.DeleteMarker.MetaSys[ReservedMetadataPrefixLower+ReplicaStatus] = []byte(string(fi.ReplicationState.ReplicaStatus))
				ventry.DeleteMarker.MetaSys[ReservedMetadataPrefixLower+ReplicaTimestamp] = []byte(fi.ReplicationState.ReplicaTimeStamp.Format(http.TimeFormat))
			default:
				ventry.DeleteMarker.MetaSys[ReservedMetadataPrefixLower+ReplicationStatus] = []byte(fi.ReplicationState.ReplicationStatusInternal)
				ventry.DeleteMarker.MetaSys[ReservedMetadataPrefixLower+ReplicationTimestamp] = []byte(fi.ReplicationState.ReplicationTimeStamp.Format(http.TimeFormat))
			}
		}
		if !fi.VersionPurgeStatus().Empty() {
			ventry.DeleteMarker.MetaSys[VersionPurgeStatusKey] = []byte(fi.ReplicationState.VersionPurgeStatusInternal)
		}
		for k, v := range fi.ReplicationState.ResetStatusesMap {
			ventry.DeleteMarker.MetaSys[k] = []byte(v)
		}
	}

	for i, ver := range x.versions {
		if ver.header.VersionID != uv {
			continue
		}
		switch ver.header.Type {
		case LegacyType:
			ver, err := x.getIdx(i)
			if err != nil {
				return "", false, err
			}
			x.versions = append(x.versions[:i], x.versions[i+1:]...)
			if fi.Deleted {
				err = x.addVersion(ventry)
			}
			return ver.ObjectV1.DataDir, len(x.versions) == 0, err
		case DeleteType:
			var err error

			if updateVersion {
				ver, err := x.getIdx(i)
				if err != nil {
					return "", false, err
				}
				if len(ver.DeleteMarker.MetaSys) == 0 {
					ver.DeleteMarker.MetaSys = make(map[string][]byte)
				}
				if !fi.DeleteMarkerReplicationStatus().Empty() {
					switch fi.DeleteMarkerReplicationStatus() {
					case replication.Replica:
						ver.DeleteMarker.MetaSys[ReservedMetadataPrefixLower+ReplicaStatus] = []byte(string(fi.ReplicationState.ReplicaStatus))
						ver.DeleteMarker.MetaSys[ReservedMetadataPrefixLower+ReplicaTimestamp] = []byte(fi.ReplicationState.ReplicaTimeStamp.Format(http.TimeFormat))
					default:
						ver.DeleteMarker.MetaSys[ReservedMetadataPrefixLower+ReplicationStatus] = []byte(fi.ReplicationState.ReplicationStatusInternal)
						ver.DeleteMarker.MetaSys[ReservedMetadataPrefixLower+ReplicationTimestamp] = []byte(fi.ReplicationState.ReplicationTimeStamp.Format(http.TimeFormat))
					}
				}
				if !fi.VersionPurgeStatus().Empty() {
					ver.DeleteMarker.MetaSys[VersionPurgeStatusKey] = []byte(fi.ReplicationState.VersionPurgeStatusInternal)
				}
				for k, v := range fi.ReplicationState.ResetStatusesMap {
					ver.DeleteMarker.MetaSys[k] = []byte(v)
				}
				err = x.setIdx(i, *ver)
			} else {
				x.versions = append(x.versions[:i], x.versions[i+1:]...)
				if fi.MarkDeleted && (fi.VersionPurgeStatus().Empty() || (fi.VersionPurgeStatus() != Complete)) {
					err = x.addVersion(ventry)
				}
			}
			return "", len(x.versions) == 0, err
		case ObjectType:
			if updateVersion {
				ver, err := x.getIdx(i)
				if err != nil {
					return "", false, err
				}
				ver.ObjectV2.MetaSys[VersionPurgeStatusKey] = []byte(fi.ReplicationState.VersionPurgeStatusInternal)
				for k, v := range fi.ReplicationState.ResetStatusesMap {
					ver.ObjectV2.MetaSys[k] = []byte(v)
				}
				err = x.setIdx(i, *ver)
				return "", len(x.versions) == 0, err
			}
		}
	}

	for i, version := range x.versions {
		if version.header.Type != ObjectType || version.header.VersionID != uv {
			continue
		}
		ver, err := x.getIdx(i)
		if err != nil {
			return "", false, err
		}
		switch {
		case fi.ExpireRestored:
			ver.ObjectV2.RemoveRestoreHdrs()
			err = x.setIdx(i, *ver)
		case fi.TransitionStatus == lifecycle.TransitionComplete:
			ver.ObjectV2.SetTransition(fi)
			err = x.setIdx(i, *ver)
		default:
			x.versions = append(x.versions[:i], x.versions[i+1:]...)
			// if uv has tiered content we add a
			// free-version to track it for
			// asynchronous deletion via scanner.
			if freeVersion, toFree := ver.ObjectV2.InitFreeVersion(fi); toFree {
				err = x.addVersion(freeVersion)
			}
		}
		logger.LogIf(context.Background(), err)

		if fi.Deleted {
			err = x.addVersion(ventry)
		}
		if x.SharedDataDirCount(ver.ObjectV2.VersionID, ver.ObjectV2.DataDir) > 0 {
			// Found that another version references the same dataDir
			// we shouldn't remove it, and only remove the version instead
			return "", len(x.versions) == 0, nil
		}
		return uuid.UUID(ver.ObjectV2.DataDir).String(), len(x.versions) == 0, err
	}

	if fi.Deleted {
		err = x.addVersion(ventry)
		return "", false, err
	}
	return "", false, errFileVersionNotFound
}

// xlMetaDataDirDecoder is a shallow decoder for decoding object datadir only.
type xlMetaDataDirDecoder struct {
	ObjectV2 *struct {
		DataDir [16]byte `msg:"DDir"` // Data dir ID
	} `msg:"V2Obj,omitempty"`
}

// UpdateObjectVersion updates metadata and modTime for a given
// versionID, NOTE: versionID must be valid and should exist -
// and must not be a DeleteMarker or legacy object, if no
// versionID is specified 'null' versionID is updated instead.
//
// It is callers responsibility to set correct versionID, this
// function shouldn't be further extended to update immutable
// values such as ErasureInfo, ChecksumInfo.
//
// Metadata is only updated to new values, existing values
// stay as is, if you wish to update all values you should
// update all metadata freshly before calling this function
// in-case you wish to clear existing metadata.
func (x *xlMetaV2Shallow) UpdateObjectVersion(fi FileInfo) error {
	if fi.VersionID == "" {
		// this means versioning is not yet
		// enabled or suspend i.e all versions
		// are basically default value i.e "null"
		fi.VersionID = nullVersionID
	}

	var uv uuid.UUID
	var err error
	if fi.VersionID != "" && fi.VersionID != nullVersionID {
		uv, err = uuid.Parse(fi.VersionID)
		if err != nil {
			return err
		}
	}

	for i, version := range x.versions {
		switch version.header.Type {
		case LegacyType, DeleteType:
			if version.header.VersionID == uv {
				return errMethodNotAllowed
			}
		case ObjectType:
			if version.header.VersionID == uv {
				ver, err := x.getIdx(i)
				if err != nil {
					return err
				}
				for k, v := range fi.Metadata {
					if strings.HasPrefix(strings.ToLower(k), ReservedMetadataPrefixLower) {
						ver.ObjectV2.MetaSys[k] = []byte(v)
					} else {
						ver.ObjectV2.MetaUser[k] = v
					}
				}
				if !fi.ModTime.IsZero() {
					ver.ObjectV2.ModTime = fi.ModTime.UnixNano()
				}
				return x.setIdx(i, *ver)
			}
		}
	}

	return errFileVersionNotFound
}

// AddVersion adds a new version
func (x *xlMetaV2Shallow) AddVersion(fi FileInfo) error {
	if fi.VersionID == "" {
		// this means versioning is not yet
		// enabled or suspend i.e all versions
		// are basically default value i.e "null"
		fi.VersionID = nullVersionID
	}

	var uv uuid.UUID
	var err error
	if fi.VersionID != "" && fi.VersionID != nullVersionID {
		uv, err = uuid.Parse(fi.VersionID)
		if err != nil {
			return err
		}
	}

	var dd uuid.UUID
	if fi.DataDir != "" {
		dd, err = uuid.Parse(fi.DataDir)
		if err != nil {
			return err
		}
	}

	ventry := xlMetaV2Version{}

	if fi.Deleted {
		ventry.Type = DeleteType
		ventry.DeleteMarker = &xlMetaV2DeleteMarker{
			VersionID: uv,
			ModTime:   fi.ModTime.UnixNano(),
			MetaSys:   make(map[string][]byte),
		}
	} else {
		ventry.Type = ObjectType
		ventry.ObjectV2 = &xlMetaV2Object{
			VersionID:          uv,
			DataDir:            dd,
			Size:               fi.Size,
			ModTime:            fi.ModTime.UnixNano(),
			ErasureAlgorithm:   ReedSolomon,
			ErasureM:           fi.Erasure.DataBlocks,
			ErasureN:           fi.Erasure.ParityBlocks,
			ErasureBlockSize:   fi.Erasure.BlockSize,
			ErasureIndex:       fi.Erasure.Index,
			BitrotChecksumAlgo: HighwayHash,
			ErasureDist:        make([]uint8, len(fi.Erasure.Distribution)),
			PartNumbers:        make([]int, len(fi.Parts)),
			PartETags:          make([]string, len(fi.Parts)),
			PartSizes:          make([]int64, len(fi.Parts)),
			PartActualSizes:    make([]int64, len(fi.Parts)),
			MetaSys:            make(map[string][]byte),
			MetaUser:           make(map[string]string, len(fi.Metadata)),
		}

		for i := range fi.Erasure.Distribution {
			ventry.ObjectV2.ErasureDist[i] = uint8(fi.Erasure.Distribution[i])
		}

		for i := range fi.Parts {
			ventry.ObjectV2.PartSizes[i] = fi.Parts[i].Size
			if fi.Parts[i].ETag != "" {
				ventry.ObjectV2.PartETags[i] = fi.Parts[i].ETag
			}
			ventry.ObjectV2.PartNumbers[i] = fi.Parts[i].Number
			ventry.ObjectV2.PartActualSizes[i] = fi.Parts[i].ActualSize
		}

		tierFVIDKey := ReservedMetadataPrefixLower + tierFVID
		tierFVMarkerKey := ReservedMetadataPrefixLower + tierFVMarker
		for k, v := range fi.Metadata {
			if strings.HasPrefix(strings.ToLower(k), ReservedMetadataPrefixLower) {
				// Skip tierFVID, tierFVMarker keys; it's used
				// only for creating free-version.
				switch k {
				case tierFVIDKey, tierFVMarkerKey:
					continue
				}

				ventry.ObjectV2.MetaSys[k] = []byte(v)
			} else {
				ventry.ObjectV2.MetaUser[k] = v
			}
		}

		// If asked to save data.
		if len(fi.Data) > 0 || fi.Size == 0 {
			x.data.replace(fi.VersionID, fi.Data)
		}

		if fi.TransitionStatus != "" {
			ventry.ObjectV2.MetaSys[ReservedMetadataPrefixLower+TransitionStatus] = []byte(fi.TransitionStatus)
		}
		if fi.TransitionedObjName != "" {
			ventry.ObjectV2.MetaSys[ReservedMetadataPrefixLower+TransitionedObjectName] = []byte(fi.TransitionedObjName)
		}
		if fi.TransitionVersionID != "" {
			ventry.ObjectV2.MetaSys[ReservedMetadataPrefixLower+TransitionedVersionID] = []byte(fi.TransitionVersionID)
		}
		if fi.TransitionTier != "" {
			ventry.ObjectV2.MetaSys[ReservedMetadataPrefixLower+TransitionTier] = []byte(fi.TransitionTier)
		}
	}

	if !ventry.Valid() {
		return errors.New("internal error: invalid version entry generated")
	}

	// Check if we should replace first.
	for i, version := range x.versions {
		if version.header.VersionID != uv {
			continue
		}
		switch version.header.Type {
		case LegacyType:
			// This would convert legacy type into new ObjectType
			// this means that we are basically purging the `null`
			// version of the object.
			return x.setIdx(i, ventry)
		case ObjectType:
			return x.setIdx(i, ventry)
		case DeleteType:
			// Allowing delete marker to replaced with proper
			// object data type as well, this is not S3 complaint
			// behavior but kept here for future flexibility.
			return x.setIdx(i, ventry)
		}
	}

	// We did not find it, add it.
	return x.addVersion(ventry)
}

func (x *xlMetaV2Shallow) SharedDataDirCount(versionID [16]byte, dataDir [16]byte) int {
	// v2 object is inlined, if it is skip dataDir share check.
	if x.data.find(uuid.UUID(versionID).String()) != nil {
		return 0
	}
	var sameDataDirCount int
	for _, version := range x.versions {
		if version.header.Type != ObjectType || version.header.VersionID == versionID || !version.header.UsesDataDir() {
			continue
		}
		var decoded xlMetaDataDirDecoder
		_, err := decoded.UnmarshalMsg(version.meta)
		if err != nil || decoded.ObjectV2 == nil || decoded.ObjectV2.DataDir != dataDir {
			continue
		}
		sameDataDirCount++
	}
	return sameDataDirCount
}

func (z *xlMetaV2Shallow) SharedDataDirCountStr(versionID, dataDir string) int {
	var (
		uv   uuid.UUID
		ddir uuid.UUID
		err  error
	)
	if versionID == nullVersionID {
		versionID = ""
	}
	if versionID != "" {
		uv, err = uuid.Parse(versionID)
		if err != nil {
			return 0
		}
	}
	ddir, err = uuid.Parse(dataDir)
	if err != nil {
		return 0
	}
	return z.SharedDataDirCount(uv, ddir)
}

// AddLegacy adds a legacy version, is only called when no prior
// versions exist, safe to use it by only one function in xl-storage(RenameData)
func (z *xlMetaV2Shallow) AddLegacy(m *xlMetaV1Object) error {
	if !m.valid() {
		return errFileCorrupt
	}
	m.VersionID = nullVersionID
	m.DataDir = legacyDataDir

	return z.addVersion(xlMetaV2Version{ObjectV1: m, Type: LegacyType})
}

// ToFileInfo converts xlMetaV2 into a common FileInfo datastructure
// for consumption across callers.
func (x xlMetaV2Shallow) ToFileInfo(volume, path, versionID string) (fi FileInfo, err error) {
	var uv uuid.UUID
	if versionID != "" && versionID != nullVersionID {
		uv, err = uuid.Parse(versionID)
		if err != nil {
			logger.LogIf(GlobalContext, fmt.Errorf("invalid versionID specified %s", versionID))
			return fi, errFileVersionNotFound
		}
	}
	var succModTime int64
	isLatest := true
	nonFreeVersions := len(x.versions)
	found := false
	for _, ver := range x.versions {
		header := &ver.header
		// skip listing free-version unless explicitly requested via versionID
		if header.FreeVersion() {
			nonFreeVersions--
			if header.VersionID != uv {
				continue
			}
		}
		if found {
			continue
		}

		// We need a specific version, skip...
		if versionID != "" && uv != header.VersionID {
			isLatest = false
			succModTime = header.ModTime
			continue
		}

		// We found what we need.
		found = true
		var version xlMetaV2Version
		if _, err := version.UnmarshalMsg(ver.meta); err != nil {
			return fi, err
		}
		if fi, err = version.ToFileInfo(volume, path); err != nil {
			return fi, err
		}
		fi.IsLatest = isLatest
		if succModTime != 0 {
			fi.SuccessorModTime = time.Unix(0, succModTime)
		}
	}
	if !found {
		if versionID == "" {
			return FileInfo{}, errFileNotFound
		}

		return FileInfo{}, errFileVersionNotFound
	}
	fi.NumVersions = nonFreeVersions
	return fi, err
}

type xlMetaBuf []byte

// ToFileInfo converts xlMetaV2 into a common FileInfo datastructure
// for consumption across callers.
func (x xlMetaBuf) ToFileInfo(volume, path, versionID string) (fi FileInfo, err error) {
	var uv uuid.UUID
	if versionID != "" && versionID != nullVersionID {
		uv, err = uuid.Parse(versionID)
		if err != nil {
			logger.LogIf(GlobalContext, fmt.Errorf("invalid versionID specified %s", versionID))
			return fi, errFileVersionNotFound
		}
	}
	versions, buf, err := decodeXlHeaders(x)
	if err != nil {
		return fi, err
	}
	var header xlMetaV2VersionHeader
	var succModTime int64
	isLatest := true
	nonFreeVersions := versions
	found := false
	err = decodeVersions(buf, versions, func(idx int, hdr, meta []byte) error {
		if _, err := header.UnmarshalMsg(hdr); err != nil {
			return err
		}

		// skip listing free-version unless explicitly requested via versionID
		if header.FreeVersion() {
			nonFreeVersions--
			if header.VersionID != uv {
				return nil
			}
		}
		if found {
			return nil
		}

		// We need a specific version, skip...
		if versionID != "" && uv != header.VersionID {
			isLatest = false
			succModTime = header.ModTime
			return nil
		}

		// We found what we need.
		found = true
		var version xlMetaV2Version
		if _, err := version.UnmarshalMsg(meta); err != nil {
			return err
		}
		if fi, err = version.ToFileInfo(volume, path); err != nil {
			return err
		}
		fi.IsLatest = isLatest
		if succModTime != 0 {
			fi.SuccessorModTime = time.Unix(0, succModTime)
		}
		return nil
	})
	if !found {
		if versionID == "" {
			return FileInfo{}, errFileNotFound
		}

		return FileInfo{}, errFileVersionNotFound
	}
	fi.NumVersions = nonFreeVersions
	return fi, err
}

// ListVersions lists current versions, and current deleted
// versions returns error for unexpected entries.
// showPendingDeletes is set to true if ListVersions needs to list objects marked deleted
// but waiting to be replicated
func (x xlMetaBuf) ListVersions(volume, path string) ([]FileInfo, error) {
	vers, buf, err := decodeXlHeaders(x)
	if err != nil {
		return nil, err
	}
	var succModTime time.Time
	isLatest := true
	dst := make([]FileInfo, 0, vers)
	var xl xlMetaV2Version
	err = decodeVersions(buf, vers, func(idx int, hdr, meta []byte) error {
		if _, err := xl.UnmarshalMsg(meta); err != nil {
			return err
		}
		if !xl.Valid() {
			return errFileCorrupt
		}
		fi, err := xl.ToFileInfo(volume, path)
		if err != nil {
			return err
		}
		fi.IsLatest = isLatest
		fi.SuccessorModTime = succModTime
		fi.NumVersions = vers
		isLatest = false
		succModTime = xl.getModTime()

		dst = append(dst, fi)
		return nil
	})
	return dst, err
}

// ListVersions lists current versions, and current deleted
// versions returns error for unexpected entries.
// showPendingDeletes is set to true if ListVersions needs to list objects marked deleted
// but waiting to be replicated
func (z xlMetaV2Shallow) ListVersions(volume, path string) ([]FileInfo, error) {
	versions := make([]FileInfo, 0, len(z.versions))
	var err error

	var dst xlMetaV2Version
	for _, version := range z.versions {
		_, err = dst.UnmarshalMsg(version.meta)
		if err != nil {
			return versions, err
		}
		fi, err := dst.ToFileInfo(volume, path)
		if err != nil {
			return versions, err
		}
		fi.NumVersions = len(z.versions)
		versions = append(versions, fi)
	}

	for i := range versions {
		versions[i].NumVersions = len(versions)
		if i > 0 {
			versions[i].SuccessorModTime = versions[i-1].ModTime
		}
	}
	if len(versions) > 0 {
		versions[0].IsLatest = true
	}
	return versions, nil
}

// IsLatestDeleteMarker returns true if latest version is a deletemarker or there are no versions.
// If any error occurs false is returned.
func (x xlMetaBuf) IsLatestDeleteMarker() bool {
	vers, buf, err := decodeXlHeaders(x)
	if err != nil {
		return false
	}
	if vers == 0 {
		return true
	}
	isDeleteMarker := false

	_ = decodeVersions(buf, vers, func(idx int, hdr, _ []byte) error {
		var xl xlMetaV2VersionHeader
		if _, err := xl.UnmarshalMsg(hdr); err != nil {
			return errDoneForNow
		}
		isDeleteMarker = xl.Type == DeleteType
		return errDoneForNow

	})
	return isDeleteMarker
}
