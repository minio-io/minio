// Code generated by "stringer -type=APIErrorCode -trimprefix=Err api-errors.go"; DO NOT EDIT.

package cmd

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[ErrNone-0]
	_ = x[ErrAccessDenied-1]
	_ = x[ErrBadDigest-2]
	_ = x[ErrEntityTooSmall-3]
	_ = x[ErrEntityTooLarge-4]
	_ = x[ErrPolicyTooLarge-5]
	_ = x[ErrIncompleteBody-6]
	_ = x[ErrInternalError-7]
	_ = x[ErrInvalidAccessKeyID-8]
	_ = x[ErrAccessKeyDisabled-9]
	_ = x[ErrInvalidBucketName-10]
	_ = x[ErrInvalidDigest-11]
	_ = x[ErrInvalidRange-12]
	_ = x[ErrInvalidRangePartNumber-13]
	_ = x[ErrInvalidCopyPartRange-14]
	_ = x[ErrInvalidCopyPartRangeSource-15]
	_ = x[ErrInvalidMaxKeys-16]
	_ = x[ErrInvalidEncodingMethod-17]
	_ = x[ErrInvalidMaxUploads-18]
	_ = x[ErrInvalidMaxParts-19]
	_ = x[ErrInvalidPartNumberMarker-20]
	_ = x[ErrInvalidPartNumber-21]
	_ = x[ErrInvalidRequestBody-22]
	_ = x[ErrInvalidCopySource-23]
	_ = x[ErrInvalidMetadataDirective-24]
	_ = x[ErrInvalidCopyDest-25]
	_ = x[ErrInvalidPolicyDocument-26]
	_ = x[ErrInvalidObjectState-27]
	_ = x[ErrMalformedXML-28]
	_ = x[ErrMissingContentLength-29]
	_ = x[ErrMissingContentMD5-30]
	_ = x[ErrMissingRequestBodyError-31]
	_ = x[ErrMissingSecurityHeader-32]
	_ = x[ErrNoSuchBucket-33]
	_ = x[ErrNoSuchBucketPolicy-34]
	_ = x[ErrNoSuchBucketLifecycle-35]
	_ = x[ErrNoSuchLifecycleConfiguration-36]
	_ = x[ErrInvalidLifecycleWithObjectLock-37]
	_ = x[ErrNoSuchBucketSSEConfig-38]
	_ = x[ErrNoSuchCORSConfiguration-39]
	_ = x[ErrNoSuchWebsiteConfiguration-40]
	_ = x[ErrReplicationConfigurationNotFoundError-41]
	_ = x[ErrRemoteDestinationNotFoundError-42]
	_ = x[ErrReplicationDestinationMissingLock-43]
	_ = x[ErrRemoteTargetNotFoundError-44]
	_ = x[ErrReplicationRemoteConnectionError-45]
	_ = x[ErrReplicationBandwidthLimitError-46]
	_ = x[ErrBucketRemoteIdenticalToSource-47]
	_ = x[ErrBucketRemoteAlreadyExists-48]
	_ = x[ErrBucketRemoteLabelInUse-49]
	_ = x[ErrBucketRemoteArnTypeInvalid-50]
	_ = x[ErrBucketRemoteArnInvalid-51]
	_ = x[ErrBucketRemoteRemoveDisallowed-52]
	_ = x[ErrRemoteTargetNotVersionedError-53]
	_ = x[ErrReplicationSourceNotVersionedError-54]
	_ = x[ErrReplicationNeedsVersioningError-55]
	_ = x[ErrReplicationBucketNeedsVersioningError-56]
	_ = x[ErrReplicationDenyEditError-57]
	_ = x[ErrReplicationNoExistingObjects-58]
	_ = x[ErrObjectRestoreAlreadyInProgress-59]
	_ = x[ErrNoSuchKey-60]
	_ = x[ErrNoSuchUpload-61]
	_ = x[ErrInvalidVersionID-62]
	_ = x[ErrNoSuchVersion-63]
	_ = x[ErrNotImplemented-64]
	_ = x[ErrPreconditionFailed-65]
	_ = x[ErrRequestTimeTooSkewed-66]
	_ = x[ErrSignatureDoesNotMatch-67]
	_ = x[ErrMethodNotAllowed-68]
	_ = x[ErrInvalidPart-69]
	_ = x[ErrInvalidPartOrder-70]
	_ = x[ErrAuthorizationHeaderMalformed-71]
	_ = x[ErrMalformedPOSTRequest-72]
	_ = x[ErrPOSTFileRequired-73]
	_ = x[ErrSignatureVersionNotSupported-74]
	_ = x[ErrBucketNotEmpty-75]
	_ = x[ErrAllAccessDisabled-76]
	_ = x[ErrMalformedPolicy-77]
	_ = x[ErrMissingFields-78]
	_ = x[ErrMissingCredTag-79]
	_ = x[ErrCredMalformed-80]
	_ = x[ErrInvalidRegion-81]
	_ = x[ErrInvalidServiceS3-82]
	_ = x[ErrInvalidServiceSTS-83]
	_ = x[ErrInvalidRequestVersion-84]
	_ = x[ErrMissingSignTag-85]
	_ = x[ErrMissingSignHeadersTag-86]
	_ = x[ErrMalformedDate-87]
	_ = x[ErrMalformedPresignedDate-88]
	_ = x[ErrMalformedCredentialDate-89]
	_ = x[ErrMalformedCredentialRegion-90]
	_ = x[ErrMalformedExpires-91]
	_ = x[ErrNegativeExpires-92]
	_ = x[ErrAuthHeaderEmpty-93]
	_ = x[ErrExpiredPresignRequest-94]
	_ = x[ErrRequestNotReadyYet-95]
	_ = x[ErrUnsignedHeaders-96]
	_ = x[ErrMissingDateHeader-97]
	_ = x[ErrInvalidQuerySignatureAlgo-98]
	_ = x[ErrInvalidQueryParams-99]
	_ = x[ErrBucketAlreadyOwnedByYou-100]
	_ = x[ErrInvalidDuration-101]
	_ = x[ErrBucketAlreadyExists-102]
	_ = x[ErrTooManyBuckets-103]
	_ = x[ErrMetadataTooLarge-104]
	_ = x[ErrUnsupportedMetadata-105]
	_ = x[ErrMaximumExpires-106]
	_ = x[ErrSlowDown-107]
	_ = x[ErrInvalidPrefixMarker-108]
	_ = x[ErrBadRequest-109]
	_ = x[ErrKeyTooLongError-110]
	_ = x[ErrInvalidBucketObjectLockConfiguration-111]
	_ = x[ErrObjectLockConfigurationNotFound-112]
	_ = x[ErrObjectLockConfigurationNotAllowed-113]
	_ = x[ErrNoSuchObjectLockConfiguration-114]
	_ = x[ErrObjectLocked-115]
	_ = x[ErrInvalidRetentionDate-116]
	_ = x[ErrPastObjectLockRetainDate-117]
	_ = x[ErrUnknownWORMModeDirective-118]
	_ = x[ErrBucketTaggingNotFound-119]
	_ = x[ErrObjectLockInvalidHeaders-120]
	_ = x[ErrInvalidTagDirective-121]
	_ = x[ErrPolicyAlreadyAttached-122]
	_ = x[ErrPolicyNotAttached-123]
	_ = x[ErrInvalidEncryptionMethod-124]
	_ = x[ErrInvalidEncryptionKeyID-125]
	_ = x[ErrInsecureSSECustomerRequest-126]
	_ = x[ErrSSEMultipartEncrypted-127]
	_ = x[ErrSSEEncryptedObject-128]
	_ = x[ErrInvalidEncryptionParameters-129]
	_ = x[ErrInvalidSSECustomerAlgorithm-130]
	_ = x[ErrInvalidSSECustomerKey-131]
	_ = x[ErrMissingSSECustomerKey-132]
	_ = x[ErrMissingSSECustomerKeyMD5-133]
	_ = x[ErrSSECustomerKeyMD5Mismatch-134]
	_ = x[ErrInvalidSSECustomerParameters-135]
	_ = x[ErrIncompatibleEncryptionMethod-136]
	_ = x[ErrKMSNotConfigured-137]
	_ = x[ErrKMSKeyNotFoundException-138]
	_ = x[ErrNoAccessKey-139]
	_ = x[ErrInvalidToken-140]
	_ = x[ErrEventNotification-141]
	_ = x[ErrARNNotification-142]
	_ = x[ErrRegionNotification-143]
	_ = x[ErrOverlappingFilterNotification-144]
	_ = x[ErrFilterNameInvalid-145]
	_ = x[ErrFilterNamePrefix-146]
	_ = x[ErrFilterNameSuffix-147]
	_ = x[ErrFilterValueInvalid-148]
	_ = x[ErrOverlappingConfigs-149]
	_ = x[ErrUnsupportedNotification-150]
	_ = x[ErrContentSHA256Mismatch-151]
	_ = x[ErrContentChecksumMismatch-152]
	_ = x[ErrReadQuorum-153]
	_ = x[ErrWriteQuorum-154]
	_ = x[ErrStorageFull-155]
	_ = x[ErrRequestBodyParse-156]
	_ = x[ErrObjectExistsAsDirectory-157]
	_ = x[ErrInvalidObjectName-158]
	_ = x[ErrInvalidObjectNamePrefixSlash-159]
	_ = x[ErrInvalidResourceName-160]
	_ = x[ErrServerNotInitialized-161]
	_ = x[ErrOperationTimedOut-162]
	_ = x[ErrClientDisconnected-163]
	_ = x[ErrOperationMaxedOut-164]
	_ = x[ErrInvalidRequest-165]
	_ = x[ErrTransitionStorageClassNotFoundError-166]
	_ = x[ErrInvalidStorageClass-167]
	_ = x[ErrBackendDown-168]
	_ = x[ErrMalformedJSON-169]
	_ = x[ErrAdminNoSuchUser-170]
	_ = x[ErrAdminNoSuchGroup-171]
	_ = x[ErrAdminGroupNotEmpty-172]
	_ = x[ErrAdminNoSuchJob-173]
	_ = x[ErrAdminNoSuchPolicy-174]
	_ = x[ErrAdminInvalidArgument-175]
	_ = x[ErrAdminInvalidAccessKey-176]
	_ = x[ErrAdminInvalidSecretKey-177]
	_ = x[ErrAdminConfigNoQuorum-178]
	_ = x[ErrAdminConfigTooLarge-179]
	_ = x[ErrAdminConfigBadJSON-180]
	_ = x[ErrAdminNoSuchConfigTarget-181]
	_ = x[ErrAdminConfigEnvOverridden-182]
	_ = x[ErrAdminConfigDuplicateKeys-183]
	_ = x[ErrAdminConfigInvalidIDPType-184]
	_ = x[ErrAdminConfigLDAPValidation-185]
	_ = x[ErrAdminCredentialsMismatch-186]
	_ = x[ErrInsecureClientRequest-187]
	_ = x[ErrObjectTampered-188]
	_ = x[ErrSiteReplicationInvalidRequest-189]
	_ = x[ErrSiteReplicationPeerResp-190]
	_ = x[ErrSiteReplicationBackendIssue-191]
	_ = x[ErrSiteReplicationServiceAccountError-192]
	_ = x[ErrSiteReplicationBucketConfigError-193]
	_ = x[ErrSiteReplicationBucketMetaError-194]
	_ = x[ErrSiteReplicationIAMError-195]
	_ = x[ErrSiteReplicationConfigMissing-196]
	_ = x[ErrAdminBucketQuotaExceeded-197]
	_ = x[ErrAdminNoSuchQuotaConfiguration-198]
	_ = x[ErrHealNotImplemented-199]
	_ = x[ErrHealNoSuchProcess-200]
	_ = x[ErrHealInvalidClientToken-201]
	_ = x[ErrHealMissingBucket-202]
	_ = x[ErrHealAlreadyRunning-203]
	_ = x[ErrHealOverlappingPaths-204]
	_ = x[ErrIncorrectContinuationToken-205]
	_ = x[ErrEmptyRequestBody-206]
	_ = x[ErrUnsupportedFunction-207]
	_ = x[ErrInvalidExpressionType-208]
	_ = x[ErrBusy-209]
	_ = x[ErrUnauthorizedAccess-210]
	_ = x[ErrExpressionTooLong-211]
	_ = x[ErrIllegalSQLFunctionArgument-212]
	_ = x[ErrInvalidKeyPath-213]
	_ = x[ErrInvalidCompressionFormat-214]
	_ = x[ErrInvalidFileHeaderInfo-215]
	_ = x[ErrInvalidJSONType-216]
	_ = x[ErrInvalidQuoteFields-217]
	_ = x[ErrInvalidRequestParameter-218]
	_ = x[ErrInvalidDataType-219]
	_ = x[ErrInvalidTextEncoding-220]
	_ = x[ErrInvalidDataSource-221]
	_ = x[ErrInvalidTableAlias-222]
	_ = x[ErrMissingRequiredParameter-223]
	_ = x[ErrObjectSerializationConflict-224]
	_ = x[ErrUnsupportedSQLOperation-225]
	_ = x[ErrUnsupportedSQLStructure-226]
	_ = x[ErrUnsupportedSyntax-227]
	_ = x[ErrUnsupportedRangeHeader-228]
	_ = x[ErrLexerInvalidChar-229]
	_ = x[ErrLexerInvalidOperator-230]
	_ = x[ErrLexerInvalidLiteral-231]
	_ = x[ErrLexerInvalidIONLiteral-232]
	_ = x[ErrParseExpectedDatePart-233]
	_ = x[ErrParseExpectedKeyword-234]
	_ = x[ErrParseExpectedTokenType-235]
	_ = x[ErrParseExpected2TokenTypes-236]
	_ = x[ErrParseExpectedNumber-237]
	_ = x[ErrParseExpectedRightParenBuiltinFunctionCall-238]
	_ = x[ErrParseExpectedTypeName-239]
	_ = x[ErrParseExpectedWhenClause-240]
	_ = x[ErrParseUnsupportedToken-241]
	_ = x[ErrParseUnsupportedLiteralsGroupBy-242]
	_ = x[ErrParseExpectedMember-243]
	_ = x[ErrParseUnsupportedSelect-244]
	_ = x[ErrParseUnsupportedCase-245]
	_ = x[ErrParseUnsupportedCaseClause-246]
	_ = x[ErrParseUnsupportedAlias-247]
	_ = x[ErrParseUnsupportedSyntax-248]
	_ = x[ErrParseUnknownOperator-249]
	_ = x[ErrParseMissingIdentAfterAt-250]
	_ = x[ErrParseUnexpectedOperator-251]
	_ = x[ErrParseUnexpectedTerm-252]
	_ = x[ErrParseUnexpectedToken-253]
	_ = x[ErrParseUnexpectedKeyword-254]
	_ = x[ErrParseExpectedExpression-255]
	_ = x[ErrParseExpectedLeftParenAfterCast-256]
	_ = x[ErrParseExpectedLeftParenValueConstructor-257]
	_ = x[ErrParseExpectedLeftParenBuiltinFunctionCall-258]
	_ = x[ErrParseExpectedArgumentDelimiter-259]
	_ = x[ErrParseCastArity-260]
	_ = x[ErrParseInvalidTypeParam-261]
	_ = x[ErrParseEmptySelect-262]
	_ = x[ErrParseSelectMissingFrom-263]
	_ = x[ErrParseExpectedIdentForGroupName-264]
	_ = x[ErrParseExpectedIdentForAlias-265]
	_ = x[ErrParseUnsupportedCallWithStar-266]
	_ = x[ErrParseNonUnaryAgregateFunctionCall-267]
	_ = x[ErrParseMalformedJoin-268]
	_ = x[ErrParseExpectedIdentForAt-269]
	_ = x[ErrParseAsteriskIsNotAloneInSelectList-270]
	_ = x[ErrParseCannotMixSqbAndWildcardInSelectList-271]
	_ = x[ErrParseInvalidContextForWildcardInSelectList-272]
	_ = x[ErrIncorrectSQLFunctionArgumentType-273]
	_ = x[ErrValueParseFailure-274]
	_ = x[ErrEvaluatorInvalidArguments-275]
	_ = x[ErrIntegerOverflow-276]
	_ = x[ErrLikeInvalidInputs-277]
	_ = x[ErrCastFailed-278]
	_ = x[ErrInvalidCast-279]
	_ = x[ErrEvaluatorInvalidTimestampFormatPattern-280]
	_ = x[ErrEvaluatorInvalidTimestampFormatPatternSymbolForParsing-281]
	_ = x[ErrEvaluatorTimestampFormatPatternDuplicateFields-282]
	_ = x[ErrEvaluatorTimestampFormatPatternHourClockAmPmMismatch-283]
	_ = x[ErrEvaluatorUnterminatedTimestampFormatPatternToken-284]
	_ = x[ErrEvaluatorInvalidTimestampFormatPatternToken-285]
	_ = x[ErrEvaluatorInvalidTimestampFormatPatternSymbol-286]
	_ = x[ErrEvaluatorBindingDoesNotExist-287]
	_ = x[ErrMissingHeaders-288]
	_ = x[ErrInvalidColumnIndex-289]
	_ = x[ErrAdminConfigNotificationTargetsFailed-290]
	_ = x[ErrAdminProfilerNotEnabled-291]
	_ = x[ErrInvalidDecompressedSize-292]
	_ = x[ErrAddUserInvalidArgument-293]
	_ = x[ErrAdminResourceInvalidArgument-294]
	_ = x[ErrAdminAccountNotEligible-295]
	_ = x[ErrAccountNotEligible-296]
	_ = x[ErrAdminServiceAccountNotFound-297]
	_ = x[ErrPostPolicyConditionInvalidFormat-298]
	_ = x[ErrInvalidChecksum-299]
}

const _APIErrorCode_name = "NoneAccessDeniedBadDigestEntityTooSmallEntityTooLargePolicyTooLargeIncompleteBodyInternalErrorInvalidAccessKeyIDAccessKeyDisabledInvalidBucketNameInvalidDigestInvalidRangeInvalidRangePartNumberInvalidCopyPartRangeInvalidCopyPartRangeSourceInvalidMaxKeysInvalidEncodingMethodInvalidMaxUploadsInvalidMaxPartsInvalidPartNumberMarkerInvalidPartNumberInvalidRequestBodyInvalidCopySourceInvalidMetadataDirectiveInvalidCopyDestInvalidPolicyDocumentInvalidObjectStateMalformedXMLMissingContentLengthMissingContentMD5MissingRequestBodyErrorMissingSecurityHeaderNoSuchBucketNoSuchBucketPolicyNoSuchBucketLifecycleNoSuchLifecycleConfigurationInvalidLifecycleWithObjectLockNoSuchBucketSSEConfigNoSuchCORSConfigurationNoSuchWebsiteConfigurationReplicationConfigurationNotFoundErrorRemoteDestinationNotFoundErrorReplicationDestinationMissingLockRemoteTargetNotFoundErrorReplicationRemoteConnectionErrorReplicationBandwidthLimitErrorBucketRemoteIdenticalToSourceBucketRemoteAlreadyExistsBucketRemoteLabelInUseBucketRemoteArnTypeInvalidBucketRemoteArnInvalidBucketRemoteRemoveDisallowedRemoteTargetNotVersionedErrorReplicationSourceNotVersionedErrorReplicationNeedsVersioningErrorReplicationBucketNeedsVersioningErrorReplicationDenyEditErrorReplicationNoExistingObjectsObjectRestoreAlreadyInProgressNoSuchKeyNoSuchUploadInvalidVersionIDNoSuchVersionNotImplementedPreconditionFailedRequestTimeTooSkewedSignatureDoesNotMatchMethodNotAllowedInvalidPartInvalidPartOrderAuthorizationHeaderMalformedMalformedPOSTRequestPOSTFileRequiredSignatureVersionNotSupportedBucketNotEmptyAllAccessDisabledMalformedPolicyMissingFieldsMissingCredTagCredMalformedInvalidRegionInvalidServiceS3InvalidServiceSTSInvalidRequestVersionMissingSignTagMissingSignHeadersTagMalformedDateMalformedPresignedDateMalformedCredentialDateMalformedCredentialRegionMalformedExpiresNegativeExpiresAuthHeaderEmptyExpiredPresignRequestRequestNotReadyYetUnsignedHeadersMissingDateHeaderInvalidQuerySignatureAlgoInvalidQueryParamsBucketAlreadyOwnedByYouInvalidDurationBucketAlreadyExistsTooManyBucketsMetadataTooLargeUnsupportedMetadataMaximumExpiresSlowDownInvalidPrefixMarkerBadRequestKeyTooLongErrorInvalidBucketObjectLockConfigurationObjectLockConfigurationNotFoundObjectLockConfigurationNotAllowedNoSuchObjectLockConfigurationObjectLockedInvalidRetentionDatePastObjectLockRetainDateUnknownWORMModeDirectiveBucketTaggingNotFoundObjectLockInvalidHeadersInvalidTagDirectivePolicyAlreadyAttachedPolicyNotAttachedInvalidEncryptionMethodInvalidEncryptionKeyIDInsecureSSECustomerRequestSSEMultipartEncryptedSSEEncryptedObjectInvalidEncryptionParametersInvalidSSECustomerAlgorithmInvalidSSECustomerKeyMissingSSECustomerKeyMissingSSECustomerKeyMD5SSECustomerKeyMD5MismatchInvalidSSECustomerParametersIncompatibleEncryptionMethodKMSNotConfiguredKMSKeyNotFoundExceptionNoAccessKeyInvalidTokenEventNotificationARNNotificationRegionNotificationOverlappingFilterNotificationFilterNameInvalidFilterNamePrefixFilterNameSuffixFilterValueInvalidOverlappingConfigsUnsupportedNotificationContentSHA256MismatchContentChecksumMismatchReadQuorumWriteQuorumStorageFullRequestBodyParseObjectExistsAsDirectoryInvalidObjectNameInvalidObjectNamePrefixSlashInvalidResourceNameServerNotInitializedOperationTimedOutClientDisconnectedOperationMaxedOutInvalidRequestTransitionStorageClassNotFoundErrorInvalidStorageClassBackendDownMalformedJSONAdminNoSuchUserAdminNoSuchGroupAdminGroupNotEmptyAdminNoSuchJobAdminNoSuchPolicyAdminInvalidArgumentAdminInvalidAccessKeyAdminInvalidSecretKeyAdminConfigNoQuorumAdminConfigTooLargeAdminConfigBadJSONAdminNoSuchConfigTargetAdminConfigEnvOverriddenAdminConfigDuplicateKeysAdminConfigInvalidIDPTypeAdminConfigLDAPValidationAdminCredentialsMismatchInsecureClientRequestObjectTamperedSiteReplicationInvalidRequestSiteReplicationPeerRespSiteReplicationBackendIssueSiteReplicationServiceAccountErrorSiteReplicationBucketConfigErrorSiteReplicationBucketMetaErrorSiteReplicationIAMErrorSiteReplicationConfigMissingAdminBucketQuotaExceededAdminNoSuchQuotaConfigurationHealNotImplementedHealNoSuchProcessHealInvalidClientTokenHealMissingBucketHealAlreadyRunningHealOverlappingPathsIncorrectContinuationTokenEmptyRequestBodyUnsupportedFunctionInvalidExpressionTypeBusyUnauthorizedAccessExpressionTooLongIllegalSQLFunctionArgumentInvalidKeyPathInvalidCompressionFormatInvalidFileHeaderInfoInvalidJSONTypeInvalidQuoteFieldsInvalidRequestParameterInvalidDataTypeInvalidTextEncodingInvalidDataSourceInvalidTableAliasMissingRequiredParameterObjectSerializationConflictUnsupportedSQLOperationUnsupportedSQLStructureUnsupportedSyntaxUnsupportedRangeHeaderLexerInvalidCharLexerInvalidOperatorLexerInvalidLiteralLexerInvalidIONLiteralParseExpectedDatePartParseExpectedKeywordParseExpectedTokenTypeParseExpected2TokenTypesParseExpectedNumberParseExpectedRightParenBuiltinFunctionCallParseExpectedTypeNameParseExpectedWhenClauseParseUnsupportedTokenParseUnsupportedLiteralsGroupByParseExpectedMemberParseUnsupportedSelectParseUnsupportedCaseParseUnsupportedCaseClauseParseUnsupportedAliasParseUnsupportedSyntaxParseUnknownOperatorParseMissingIdentAfterAtParseUnexpectedOperatorParseUnexpectedTermParseUnexpectedTokenParseUnexpectedKeywordParseExpectedExpressionParseExpectedLeftParenAfterCastParseExpectedLeftParenValueConstructorParseExpectedLeftParenBuiltinFunctionCallParseExpectedArgumentDelimiterParseCastArityParseInvalidTypeParamParseEmptySelectParseSelectMissingFromParseExpectedIdentForGroupNameParseExpectedIdentForAliasParseUnsupportedCallWithStarParseNonUnaryAgregateFunctionCallParseMalformedJoinParseExpectedIdentForAtParseAsteriskIsNotAloneInSelectListParseCannotMixSqbAndWildcardInSelectListParseInvalidContextForWildcardInSelectListIncorrectSQLFunctionArgumentTypeValueParseFailureEvaluatorInvalidArgumentsIntegerOverflowLikeInvalidInputsCastFailedInvalidCastEvaluatorInvalidTimestampFormatPatternEvaluatorInvalidTimestampFormatPatternSymbolForParsingEvaluatorTimestampFormatPatternDuplicateFieldsEvaluatorTimestampFormatPatternHourClockAmPmMismatchEvaluatorUnterminatedTimestampFormatPatternTokenEvaluatorInvalidTimestampFormatPatternTokenEvaluatorInvalidTimestampFormatPatternSymbolEvaluatorBindingDoesNotExistMissingHeadersInvalidColumnIndexAdminConfigNotificationTargetsFailedAdminProfilerNotEnabledInvalidDecompressedSizeAddUserInvalidArgumentAdminResourceInvalidArgumentAdminAccountNotEligibleAccountNotEligibleAdminServiceAccountNotFoundPostPolicyConditionInvalidFormatInvalidChecksum"

var _APIErrorCode_index = [...]uint16{0, 4, 16, 25, 39, 53, 67, 81, 94, 112, 129, 146, 159, 171, 193, 213, 239, 253, 274, 291, 306, 329, 346, 364, 381, 405, 420, 441, 459, 471, 491, 508, 531, 552, 564, 582, 603, 631, 661, 682, 705, 731, 768, 798, 831, 856, 888, 918, 947, 972, 994, 1020, 1042, 1070, 1099, 1133, 1164, 1201, 1225, 1253, 1283, 1292, 1304, 1320, 1333, 1347, 1365, 1385, 1406, 1422, 1433, 1449, 1477, 1497, 1513, 1541, 1555, 1572, 1587, 1600, 1614, 1627, 1640, 1656, 1673, 1694, 1708, 1729, 1742, 1764, 1787, 1812, 1828, 1843, 1858, 1879, 1897, 1912, 1929, 1954, 1972, 1995, 2010, 2029, 2043, 2059, 2078, 2092, 2100, 2119, 2129, 2144, 2180, 2211, 2244, 2273, 2285, 2305, 2329, 2353, 2374, 2398, 2417, 2438, 2455, 2478, 2500, 2526, 2547, 2565, 2592, 2619, 2640, 2661, 2685, 2710, 2738, 2766, 2782, 2805, 2816, 2828, 2845, 2860, 2878, 2907, 2924, 2940, 2956, 2974, 2992, 3015, 3036, 3059, 3069, 3080, 3091, 3107, 3130, 3147, 3175, 3194, 3214, 3231, 3249, 3266, 3280, 3315, 3334, 3345, 3358, 3373, 3389, 3407, 3421, 3438, 3458, 3479, 3500, 3519, 3538, 3556, 3579, 3603, 3627, 3652, 3677, 3701, 3722, 3736, 3765, 3788, 3815, 3849, 3881, 3911, 3934, 3962, 3986, 4015, 4033, 4050, 4072, 4089, 4107, 4127, 4153, 4169, 4188, 4209, 4213, 4231, 4248, 4274, 4288, 4312, 4333, 4348, 4366, 4389, 4404, 4423, 4440, 4457, 4481, 4508, 4531, 4554, 4571, 4593, 4609, 4629, 4648, 4670, 4691, 4711, 4733, 4757, 4776, 4818, 4839, 4862, 4883, 4914, 4933, 4955, 4975, 5001, 5022, 5044, 5064, 5088, 5111, 5130, 5150, 5172, 5195, 5226, 5264, 5305, 5335, 5349, 5370, 5386, 5408, 5438, 5464, 5492, 5525, 5543, 5566, 5601, 5641, 5683, 5715, 5732, 5757, 5772, 5789, 5799, 5810, 5848, 5902, 5948, 6000, 6048, 6091, 6135, 6163, 6177, 6195, 6231, 6254, 6277, 6299, 6327, 6350, 6368, 6395, 6427, 6442}

func (i APIErrorCode) String() string {
	if i < 0 || i >= APIErrorCode(len(_APIErrorCode_index)-1) {
		return "APIErrorCode(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _APIErrorCode_name[_APIErrorCode_index[i]:_APIErrorCode_index[i+1]]
}
