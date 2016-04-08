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
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/minio/minio/pkg/mimedb"
	"github.com/minio/minio/pkg/probe"
	"github.com/minio/minio/pkg/safe"
)

type objectAPI struct {
	storage StorageAPI
}

func newObjectLayer(storage StorageAPI) *objectAPI {
	return &objectAPI{storage}
}

/// Bucket operations

// MakeBucket - make a bucket.
func (o objectAPI) MakeBucket(bucket string) *probe.Error {
	// Verify if bucket is valid.
	if !IsValidBucketName(bucket) {
		return probe.NewError(BucketNameInvalid{Bucket: bucket})
	}
	if e := o.storage.MakeVol(bucket); e != nil {
		if e == errVolumeExists {
			return probe.NewError(BucketExists{Bucket: bucket})
		}
		return probe.NewError(e)
	}
	return nil
}

// GetBucketInfo - get bucket info.
func (o objectAPI) GetBucketInfo(bucket string) (BucketInfo, *probe.Error) {
	// Verify if bucket is valid.
	if !IsValidBucketName(bucket) {
		return BucketInfo{}, probe.NewError(BucketNameInvalid{Bucket: bucket})
	}
	vi, e := o.storage.StatVol(bucket)
	if e != nil {
		if e == errVolumeNotFound {
			return BucketInfo{}, probe.NewError(BucketNotFound{Bucket: bucket})
		}
		return BucketInfo{}, probe.NewError(e)
	}
	return BucketInfo{
		Name:    vi.Name,
		Created: vi.Created,
	}, nil
}

// ListBuckets - list buckets.
func (o objectAPI) ListBuckets() ([]BucketInfo, *probe.Error) {
	var bucketInfos []BucketInfo
	vols, e := o.storage.ListVols()
	if e != nil {
		return nil, probe.NewError(e)
	}
	for _, vol := range vols {
		bucketInfos = append(bucketInfos, BucketInfo{vol.Name, vol.Created})
	}
	return bucketInfos, nil
}

// DeleteBucket - delete a bucket.
func (o objectAPI) DeleteBucket(bucket string) *probe.Error {
	// Verify if bucket is valid.
	if !IsValidBucketName(bucket) {
		return probe.NewError(BucketNameInvalid{Bucket: bucket})
	}
	if e := o.storage.DeleteVol(bucket); e != nil {
		if e == errVolumeNotFound {
			return probe.NewError(BucketNotFound{Bucket: bucket})
		}
		return probe.NewError(e)
	}
	return nil
}

/// Object Operations

// GetObject - get an object.
func (o objectAPI) GetObject(bucket, object string, startOffset int64) (io.ReadCloser, *probe.Error) {
	// Verify if bucket is valid.
	if !IsValidBucketName(bucket) {
		return nil, probe.NewError(BucketNameInvalid{Bucket: bucket})
	}
	// Verify if object is valid.
	if !IsValidObjectName(object) {
		return nil, probe.NewError(ObjectNameInvalid{Bucket: bucket, Object: object})
	}
	r, e := o.storage.ReadFile(bucket, object, startOffset)
	if e != nil {
		if e == errVolumeNotFound {
			return nil, probe.NewError(BucketNotFound{Bucket: bucket})
		} else if e == errFileNotFound {
			return nil, probe.NewError(ObjectNotFound{Bucket: bucket, Object: object})
		}
		return nil, probe.NewError(e)
	}
	return r, nil
}

// GetObjectInfo - get object info.
func (o objectAPI) GetObjectInfo(bucket, object string) (ObjectInfo, *probe.Error) {
	// Verify if bucket is valid.
	if !IsValidBucketName(bucket) {
		return ObjectInfo{}, probe.NewError(BucketNameInvalid{Bucket: bucket})
	}
	// Verify if object is valid.
	if !IsValidObjectName(object) {
		return ObjectInfo{}, probe.NewError(ObjectNameInvalid{Bucket: bucket, Object: object})
	}
	fi, e := o.storage.StatFile(bucket, object)
	if e != nil {
		if e == errVolumeNotFound {
			return ObjectInfo{}, probe.NewError(BucketNotFound{Bucket: bucket})
		} else if e == errFileNotFound || e == errIsNotRegular {
			return ObjectInfo{}, probe.NewError(ObjectNotFound{Bucket: bucket, Object: object})
			// Handle more lower level errors if needed.
		} else {
			return ObjectInfo{}, probe.NewError(e)
		}
	}
	contentType := "application/octet-stream"
	if objectExt := filepath.Ext(object); objectExt != "" {
		content, ok := mimedb.DB[strings.ToLower(strings.TrimPrefix(objectExt, "."))]
		if ok {
			contentType = content.ContentType
		}
	}
	return ObjectInfo{
		Bucket:      fi.Volume,
		Name:        fi.Name,
		ModTime:     fi.ModTime,
		Size:        fi.Size,
		IsDir:       fi.Mode.IsDir(),
		ContentType: contentType,
		MD5Sum:      "", // Read from metadata.
	}, nil
}

func (o objectAPI) PutObject(bucket string, object string, size int64, data io.Reader, metadata map[string]string) (ObjectInfo, *probe.Error) {
	// Verify if bucket is valid.
	if !IsValidBucketName(bucket) {
		return ObjectInfo{}, probe.NewError(BucketNameInvalid{Bucket: bucket})
	}
	if !IsValidObjectName(object) {
		return ObjectInfo{}, probe.NewError(ObjectNameInvalid{
			Bucket: bucket,
			Object: object,
		})
	}
	fileWriter, e := o.storage.CreateFile(bucket, object)
	if e != nil {
		if e == errVolumeNotFound {
			return ObjectInfo{}, probe.NewError(BucketNotFound{
				Bucket: bucket,
			})
		} else if e == errIsNotRegular {
			return ObjectInfo{}, probe.NewError(ObjectExistsAsPrefix{
				Bucket: bucket,
				Prefix: object,
			})
		}
		return ObjectInfo{}, probe.NewError(e)
	}

	// Initialize md5 writer.
	md5Writer := md5.New()

	// Instantiate a new multi writer.
	multiWriter := io.MultiWriter(md5Writer, fileWriter)

	// Instantiate checksum hashers and create a multiwriter.
	if size > 0 {
		if _, e = io.CopyN(multiWriter, data, size); e != nil {
			fileWriter.(*safe.File).CloseAndRemove()
			return ObjectInfo{}, probe.NewError(e)
		}
	} else {
		if _, e = io.Copy(multiWriter, data); e != nil {
			fileWriter.(*safe.File).CloseAndRemove()
			return ObjectInfo{}, probe.NewError(e)
		}
	}

	newMD5Hex := hex.EncodeToString(md5Writer.Sum(nil))
	// md5Hex representation.
	var md5Hex string
	if len(metadata) != 0 {
		md5Hex = metadata["md5Sum"]
	}
	if md5Hex != "" {
		if newMD5Hex != md5Hex {
			fileWriter.(*safe.File).CloseAndRemove()
			return ObjectInfo{}, probe.NewError(BadDigest{md5Hex, newMD5Hex})
		}
	}
	e = fileWriter.Close()
	if e != nil {
		return ObjectInfo{}, probe.NewError(e)
	}
	fi, e := o.storage.StatFile(bucket, object)
	if e != nil {
		return ObjectInfo{}, probe.NewError(e)
	}

	contentType := "application/octet-stream"
	if objectExt := filepath.Ext(object); objectExt != "" {
		content, ok := mimedb.DB[strings.ToLower(strings.TrimPrefix(objectExt, "."))]
		if ok {
			contentType = content.ContentType
		}
	}

	return ObjectInfo{
		Bucket:      fi.Volume,
		Name:        fi.Name,
		ModTime:     fi.ModTime,
		Size:        fi.Size,
		ContentType: contentType,
		MD5Sum:      newMD5Hex,
	}, nil
}

func (o objectAPI) DeleteObject(bucket, object string) *probe.Error {
	// Verify if bucket is valid.
	if !IsValidBucketName(bucket) {
		return probe.NewError(BucketNameInvalid{Bucket: bucket})
	}
	if !IsValidObjectName(object) {
		return probe.NewError(ObjectNameInvalid{Bucket: bucket, Object: object})
	}
	if e := o.storage.DeleteFile(bucket, object); e != nil {
		if e == errVolumeNotFound {
			return probe.NewError(BucketNotFound{Bucket: bucket})
		}
		return probe.NewError(e)
	}
	return nil
}

func (o objectAPI) ListObjects(bucket, prefix, marker, delimiter string, maxKeys int) (ListObjectsInfo, *probe.Error) {
	// Verify if bucket is valid.
	if !IsValidBucketName(bucket) {
		return ListObjectsInfo{}, probe.NewError(BucketNameInvalid{Bucket: bucket})
	}
	if !IsValidObjectPrefix(prefix) {
		return ListObjectsInfo{}, probe.NewError(ObjectNameInvalid{Bucket: bucket, Object: prefix})
	}
	// Verify if delimiter is anything other than '/', which we do not support.
	if delimiter != "" && delimiter != "/" {
		return ListObjectsInfo{}, probe.NewError(fmt.Errorf("delimiter '%s' is not supported. Only '/' is supported", delimiter))
	}
	// Verify if marker has prefix.
	if marker != "" {
		if !strings.HasPrefix(marker, prefix) {
			return ListObjectsInfo{}, probe.NewError(fmt.Errorf("Invalid combination of marker '%s' and prefix '%s'", marker, prefix))
		}
	}
	recursive := true
	if delimiter == "/" {
		recursive = false
	}
	fileInfos, eof, e := o.storage.ListFiles(bucket, prefix, marker, recursive, maxKeys)
	if e != nil {
		if e == errVolumeNotFound {
			return ListObjectsInfo{}, probe.NewError(BucketNotFound{Bucket: bucket})
		}
		return ListObjectsInfo{}, probe.NewError(e)
	}
	if maxKeys == 0 {
		return ListObjectsInfo{}, nil
	}
	result := ListObjectsInfo{IsTruncated: !eof}
	for _, fileInfo := range fileInfos {
		result.NextMarker = fileInfo.Name
		if fileInfo.Mode.IsDir() {
			result.Prefixes = append(result.Prefixes, fileInfo.Name)
			continue
		}
		result.Objects = append(result.Objects, ObjectInfo{
			Name:    fileInfo.Name,
			ModTime: fileInfo.ModTime,
			Size:    fileInfo.Size,
			IsDir:   fileInfo.Mode.IsDir(),
		})
	}
	return result, nil
}

func (o objectAPI) ListMultipartUploads(bucket, objectPrefix, keyMarker, uploadIDMarker, delimiter string, maxUploads int) (ListMultipartsInfo, *probe.Error) {
	// Verify if bucket is valid.
	if !IsValidBucketName(bucket) {
		return ListMultipartsInfo{}, probe.NewError(BucketNameInvalid{Bucket: bucket})
	}
	if !IsValidObjectPrefix(objectPrefix) {
		return ListMultipartsInfo{}, probe.NewError(ObjectNameInvalid{Bucket: bucket, Object: objectPrefix})
	}
	return ListMultipartsInfo{}, probe.NewError(errors.New("Not implemented"))
}

func (o objectAPI) NewMultipartUpload(bucket, object string) (string, *probe.Error) {
	// Verify if bucket is valid.
	if !IsValidBucketName(bucket) {
		return "", probe.NewError(BucketNameInvalid{Bucket: bucket})
	}
	if !IsValidObjectName(object) {
		return "", probe.NewError(ObjectNameInvalid{Bucket: bucket, Object: object})
	}
	return "", probe.NewError(errors.New("Not implemented"))
}

func (o objectAPI) PutObjectPart(bucket, object, uploadID string, partID int, size int64, data io.Reader, md5Hex string) (string, *probe.Error) {
	// Verify if bucket is valid.
	if !IsValidBucketName(bucket) {
		return "", probe.NewError(BucketNameInvalid{Bucket: bucket})
	}
	if !IsValidObjectName(object) {
		return "", probe.NewError(ObjectNameInvalid{Bucket: bucket, Object: object})
	}
	return "", probe.NewError(errors.New("Not implemented"))
}

func (o objectAPI) ListObjectParts(bucket, object, uploadID string, partNumberMarker, maxParts int) (ListPartsInfo, *probe.Error) {
	// Verify if bucket is valid.
	if !IsValidBucketName(bucket) {
		return ListPartsInfo{}, probe.NewError(BucketNameInvalid{Bucket: bucket})
	}
	if !IsValidObjectName(object) {
		return ListPartsInfo{}, probe.NewError(ObjectNameInvalid{Bucket: bucket, Object: object})
	}
	return ListPartsInfo{}, probe.NewError(errors.New("Not implemented"))
}

func (o objectAPI) CompleteMultipartUpload(bucket string, object string, uploadID string, parts []completePart) (ObjectInfo, *probe.Error) {
	// Verify if bucket is valid.
	if !IsValidBucketName(bucket) {
		return ObjectInfo{}, probe.NewError(BucketNameInvalid{Bucket: bucket})
	}
	if !IsValidObjectName(object) {
		return ObjectInfo{}, probe.NewError(ObjectNameInvalid{Bucket: bucket, Object: object})
	}
	return ObjectInfo{}, probe.NewError(errors.New("Not implemented"))
}

func (o objectAPI) AbortMultipartUpload(bucket, object, uploadID string) *probe.Error {
	// Verify if bucket is valid.
	if !IsValidBucketName(bucket) {
		return probe.NewError(BucketNameInvalid{Bucket: bucket})
	}
	if !IsValidObjectName(object) {
		return probe.NewError(ObjectNameInvalid{Bucket: bucket, Object: object})
	}
	return probe.NewError(errors.New("Not implemented"))
}
