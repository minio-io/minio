// Code generated by "stringer -type=storageMetric -trimprefix=storageMetric xl-storage-disk-id-check.go"; DO NOT EDIT.

package cmd

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[storageMetricMakeVolBulk-0]
	_ = x[storageMetricMakeVol-1]
	_ = x[storageMetricListVols-2]
	_ = x[storageMetricStatVol-3]
	_ = x[storageMetricDeleteVol-4]
	_ = x[storageMetricWalkDir-5]
	_ = x[storageMetricListDir-6]
	_ = x[storageMetricReadFile-7]
	_ = x[storageMetricAppendFile-8]
	_ = x[storageMetricCreateFile-9]
	_ = x[storageMetricReadFileStream-10]
	_ = x[storageMetricRenameFile-11]
	_ = x[storageMetricRenameData-12]
	_ = x[storageMetricCheckParts-13]
	_ = x[storageMetricDelete-14]
	_ = x[storageMetricDeleteVersions-15]
	_ = x[storageMetricVerifyFile-16]
	_ = x[storageMetricWriteAll-17]
	_ = x[storageMetricDeleteVersion-18]
	_ = x[storageMetricWriteMetadata-19]
	_ = x[storageMetricUpdateMetadata-20]
	_ = x[storageMetricReadVersion-21]
	_ = x[storageMetricReadAll-22]
	_ = x[storageStatInfoFile-23]
	_ = x[storageMetricLast-24]
}

const _storageMetric_name = "MakeVolBulkMakeVolListVolsStatVolDeleteVolWalkDirListDirReadFileAppendFileCreateFileReadFileStreamRenameFileRenameDataCheckPartsDeleteDeleteVersionsVerifyFileWriteAllDeleteVersionWriteMetadataUpdateMetadataReadVersionReadAllstorageStatInfoFileLast"

var _storageMetric_index = [...]uint8{0, 11, 18, 26, 33, 42, 49, 56, 64, 74, 84, 98, 108, 118, 128, 134, 148, 158, 166, 179, 192, 206, 217, 224, 243, 247}

func (i storageMetric) String() string {
	if i >= storageMetric(len(_storageMetric_index)-1) {
		return "storageMetric(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _storageMetric_name[_storageMetric_index[i]:_storageMetric_index[i+1]]
}
