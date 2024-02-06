// Code generated by "stringer -type=HandlerID -output=handlers_string.go -trimprefix=Handler msg.go handlers.go"; DO NOT EDIT.

package grid

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[handlerInvalid-0]
	_ = x[HandlerLockLock-1]
	_ = x[HandlerLockRLock-2]
	_ = x[HandlerLockUnlock-3]
	_ = x[HandlerLockRUnlock-4]
	_ = x[HandlerLockRefresh-5]
	_ = x[HandlerLockForceUnlock-6]
	_ = x[HandlerWalkDir-7]
	_ = x[HandlerStatVol-8]
	_ = x[HandlerDiskInfo-9]
	_ = x[HandlerNSScanner-10]
	_ = x[HandlerReadXL-11]
	_ = x[HandlerReadVersion-12]
	_ = x[HandlerDeleteFile-13]
	_ = x[HandlerDeleteVersion-14]
	_ = x[HandlerUpdateMetadata-15]
	_ = x[HandlerWriteMetadata-16]
	_ = x[HandlerCheckParts-17]
	_ = x[HandlerRenameData-18]
	_ = x[HandlerRenameFile-19]
	_ = x[HandlerReadAll-20]
	_ = x[HandlerServerVerify-21]
	_ = x[HandlerTrace-22]
	_ = x[HandlerListen-23]
	_ = x[HandlerGetLocalDiskIDs-24]
	_ = x[HandlerDeleteBucketMetadata-25]
	_ = x[HandlerLoadBucketMetadata-26]
	_ = x[HandlerReloadSiteReplicationConfig-27]
	_ = x[HandlerReloadPoolMeta-28]
	_ = x[HandlerStopRebalance-29]
	_ = x[HandlerLoadRebalanceMeta-30]
	_ = x[HandlerLoadTransitionTierConfig-31]
	_ = x[HandlerDeletePolicy-32]
	_ = x[HandlerLoadPolicy-33]
	_ = x[HandlerLoadPolicyMapping-34]
	_ = x[HandlerDeleteServiceAccount-35]
	_ = x[HandlerLoadServiceAccount-36]
	_ = x[HandlerDeleteUser-37]
	_ = x[HandlerLoadUser-38]
	_ = x[HandlerLoadGroup-39]
	_ = x[HandlerHealBucket-40]
	_ = x[HandlerMakeBucket-41]
	_ = x[HandlerHeadBucket-42]
	_ = x[HandlerDeleteBucket-43]
	_ = x[HandlerGetMetrics-44]
	_ = x[HandlerGetResourceMetrics-45]
	_ = x[HandlerGetMemInfo-46]
	_ = x[HandlerGetProcInfo-47]
	_ = x[HandlerGetOSInfo-48]
	_ = x[HandlerGetPartitions-49]
	_ = x[HandlerGetNetInfo-50]
	_ = x[HandlerGetCPUs-51]
	_ = x[HandlerServerInfo-52]
	_ = x[HandlerGetSysConfig-53]
	_ = x[HandlerGetSysServices-54]
	_ = x[HandlerGetSysErrors-55]
	_ = x[HandlerGetAllBucketStats-56]
	_ = x[HandlerGetBucketStats-57]
	_ = x[HandlerGetSRMetrics-58]
	_ = x[HandlerGetPeerMetrics-59]
	_ = x[HandlerGetMetacacheListing-60]
	_ = x[HandlerUpdateMetacacheListing-61]
	_ = x[HandlerGetPeerBucketMetrics-62]
	_ = x[HandlerStorageInfo-63]
	_ = x[handlerTest-64]
	_ = x[handlerTest2-65]
	_ = x[handlerLast-66]
}

const _HandlerID_name = "handlerInvalidLockLockLockRLockLockUnlockLockRUnlockLockRefreshLockForceUnlockWalkDirStatVolDiskInfoNSScannerReadXLReadVersionDeleteFileDeleteVersionUpdateMetadataWriteMetadataCheckPartsRenameDataRenameFileReadAllServerVerifyTraceListenGetLocalDiskIDsDeleteBucketMetadataLoadBucketMetadataReloadSiteReplicationConfigReloadPoolMetaStopRebalanceLoadRebalanceMetaLoadTransitionTierConfigDeletePolicyLoadPolicyLoadPolicyMappingDeleteServiceAccountLoadServiceAccountDeleteUserLoadUserLoadGroupHealBucketMakeBucketHeadBucketDeleteBucketGetMetricsGetResourceMetricsGetMemInfoGetProcInfoGetOSInfoGetPartitionsGetNetInfoGetCPUsServerInfoGetSysConfigGetSysServicesGetSysErrorsGetAllBucketStatsGetBucketStatsGetSRMetricsGetPeerMetricsGetMetacacheListingUpdateMetacacheListingGetPeerBucketMetricsStorageInfohandlerTesthandlerTest2handlerLast"

var _HandlerID_index = [...]uint16{0, 14, 22, 31, 41, 52, 63, 78, 85, 92, 100, 109, 115, 126, 136, 149, 163, 176, 186, 196, 206, 213, 225, 230, 236, 251, 271, 289, 316, 330, 343, 360, 384, 396, 406, 423, 443, 461, 471, 479, 488, 498, 508, 518, 530, 540, 558, 568, 579, 588, 601, 611, 618, 628, 640, 654, 666, 683, 697, 709, 723, 742, 764, 784, 795, 806, 818, 829}

func (i HandlerID) String() string {
	if i >= HandlerID(len(_HandlerID_index)-1) {
		return "HandlerID(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _HandlerID_name[_HandlerID_index[i]:_HandlerID_index[i+1]]
}
