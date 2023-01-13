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
	"fmt"
	"github.com/gorilla/mux"
	"github.com/klauspost/compress/gzhttp"
	"github.com/klauspost/compress/gzip"
	"github.com/minio/madmin-go"
	"github.com/minio/minio/internal/logger"
	"net/http"
	"strings"
)

const (
	adminPathPrefix       = minioReservedBucketPath + "/admin"
	adminAPIVersion       = madmin.AdminAPIVersion
	adminAPIVersionPrefix = SlashSeparator + adminAPIVersion
)

// adminAPIHandlers provides HTTP handlers for MinIO admin API.
type adminAPIHandlers struct{}

// adminApiHostHandler - allow access to the  Admin APIs only from local interface by default.
func adminApiHostHandler(f func(w http.ResponseWriter, r *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if globalIsGateway && globalGatewayName == PANFSBackendGateway && globalPanOnlyLocalAdminApi {
			host := strings.Split(r.Host, ":")[0]
			if host == "localhost" || host == "127.0.0.1" {
				f(w, r)
			} else {
				writeErrorResponse(r.Context(), w, APIError{
					Code:           "Forbidden",
					Description:    fmt.Sprintf("Admin API allowed from localhost only by default"),
					HTTPStatusCode: http.StatusForbidden,
				}, r.URL)
			}
		} else {
			f(w, r)
		}
	}
}

// registerAdminRouter - Add handler functions for each service REST API routes.
func registerAdminRouter(router *mux.Router, enableConfigOps bool) {
	adminAPI := adminAPIHandlers{}
	// Admin router
	adminRouter := router.PathPrefix(adminPathPrefix).Subrouter()

	adminVersions := []string{
		adminAPIVersionPrefix,
	}

	gz, err := gzhttp.NewWrapper(gzhttp.MinSize(1000), gzhttp.CompressionLevel(gzip.BestSpeed))
	if err != nil {
		// Static params, so this is very unlikely.
		logger.Fatal(err, "Unable to initialize server")
	}

	for _, adminVersion := range adminVersions {
		// Restart and stop MinIO service.
		adminRouter.Methods(http.MethodPost).Path(adminVersion+"/service").
			HandlerFunc(adminApiHostHandler(gz(httpTraceAll(adminAPI.ServiceHandler)))).
			Queries("action", "{action:.*}")
		// Update MinIO servers.
		adminRouter.Methods(http.MethodPost).Path(adminVersion+"/update").
			HandlerFunc(adminApiHostHandler(gz(httpTraceAll(adminAPI.ServerUpdateHandler)))).
			Queries("updateURL", "{updateURL:.*}")

		// Info operations
		adminRouter.Methods(http.MethodGet).Path(adminVersion + "/info").
			HandlerFunc(adminApiHostHandler(gz(httpTraceAll(adminAPI.ServerInfoHandler))))
		adminRouter.Methods(http.MethodGet).Path(adminVersion+"/inspect-data").
			HandlerFunc(adminApiHostHandler(httpTraceHdrs(adminAPI.InspectDataHandler))).
			Queries("volume", "{volume:.*}", "file", "{file:.*}")

		// StorageInfo operations
		adminRouter.Methods(http.MethodGet).Path(adminVersion + "/storageinfo").
			HandlerFunc(adminApiHostHandler(gz(httpTraceAll(adminAPI.StorageInfoHandler))))
		// DataUsageInfo operations
		adminRouter.Methods(http.MethodGet).Path(adminVersion + "/datausageinfo").
			HandlerFunc(adminApiHostHandler(gz(httpTraceAll(adminAPI.DataUsageInfoHandler))))
		// Metrics operation
		adminRouter.Methods(http.MethodGet).Path(adminVersion + "/metrics").
			HandlerFunc(adminApiHostHandler(gz(httpTraceAll(adminAPI.MetricsHandler))))

		if globalIsDistErasure || globalIsErasure {
			// Heal operations

			// Heal processing endpoint.
			adminRouter.Methods(http.MethodPost).Path(adminVersion + "/heal/").
				HandlerFunc(adminApiHostHandler(gz(httpTraceAll(adminAPI.HealHandler))))
			adminRouter.Methods(http.MethodPost).Path(adminVersion + "/heal/{bucket}").
				HandlerFunc(adminApiHostHandler(gz(httpTraceAll(adminAPI.HealHandler))))
			adminRouter.Methods(http.MethodPost).Path(adminVersion + "/heal/{bucket}/{prefix:.*}").
				HandlerFunc(adminApiHostHandler(gz(httpTraceAll(adminAPI.HealHandler))))
			adminRouter.Methods(http.MethodPost).Path(adminVersion + "/background-heal/status").
				HandlerFunc(adminApiHostHandler(gz(httpTraceAll(adminAPI.BackgroundHealStatusHandler))))

			// Pool operations
			adminRouter.Methods(http.MethodGet).Path(adminVersion + "/pools/list").
				HandlerFunc(adminApiHostHandler(gz(httpTraceAll(adminAPI.ListPools))))
			adminRouter.Methods(http.MethodGet).Path(adminVersion+"/pools/status").
				HandlerFunc(adminApiHostHandler(gz(httpTraceAll(adminAPI.StatusPool)))).
				Queries("pool", "{pool:.*}")

			adminRouter.Methods(http.MethodPost).Path(adminVersion+"/pools/decommission").
				HandlerFunc(adminApiHostHandler(gz(httpTraceAll(adminAPI.StartDecommission)))).
				Queries("pool", "{pool:.*}")
			adminRouter.Methods(http.MethodPost).Path(adminVersion+"/pools/cancel").
				HandlerFunc(adminApiHostHandler(gz(httpTraceAll(adminAPI.CancelDecommission)))).
				Queries("pool", "{pool:.*}")
		}

		// Profiling operations - deprecated API
		adminRouter.Methods(http.MethodPost).Path(adminVersion+"/profiling/start").
			HandlerFunc(adminApiHostHandler(gz(httpTraceAll(adminAPI.StartProfilingHandler)))).
			Queries("profilerType", "{profilerType:.*}")
		adminRouter.Methods(http.MethodGet).Path(adminVersion + "/profiling/download").
			HandlerFunc(adminApiHostHandler(gz(httpTraceAll(adminAPI.DownloadProfilingHandler))))
		// Profiling operations
		adminRouter.Methods(http.MethodPost).Path(adminVersion + "/profile").
			HandlerFunc(adminApiHostHandler(gz(httpTraceAll(adminAPI.ProfileHandler))))

		// Config KV operations.
		if enableConfigOps {
			adminRouter.Methods(http.MethodGet).Path(adminVersion+"/get-config-kv").
				HandlerFunc(adminApiHostHandler(gz(httpTraceHdrs(adminAPI.GetConfigKVHandler)))).
				Queries("key", "{key:.*}")
			adminRouter.Methods(http.MethodPut).Path(adminVersion + "/set-config-kv").
				HandlerFunc(adminApiHostHandler(gz(httpTraceHdrs(adminAPI.SetConfigKVHandler))))
			adminRouter.Methods(http.MethodDelete).Path(adminVersion + "/del-config-kv").
				HandlerFunc(adminApiHostHandler(gz(httpTraceHdrs(adminAPI.DelConfigKVHandler))))
		}

		// Enable config help in all modes.
		adminRouter.Methods(http.MethodGet).Path(adminVersion+"/help-config-kv").
			HandlerFunc(adminApiHostHandler(gz(httpTraceAll(adminAPI.HelpConfigKVHandler)))).
			Queries("subSys", "{subSys:.*}", "key", "{key:.*}")

		// Config KV history operations.
		if enableConfigOps {
			adminRouter.Methods(http.MethodGet).Path(adminVersion+"/list-config-history-kv").
				HandlerFunc(adminApiHostHandler(gz(httpTraceAll(adminAPI.ListConfigHistoryKVHandler)))).
				Queries("count", "{count:[0-9]+}")
			adminRouter.Methods(http.MethodDelete).Path(adminVersion+"/clear-config-history-kv").
				HandlerFunc(adminApiHostHandler(gz(httpTraceHdrs(adminAPI.ClearConfigHistoryKVHandler)))).
				Queries("restoreId", "{restoreId:.*}")
			adminRouter.Methods(http.MethodPut).Path(adminVersion+"/restore-config-history-kv").
				HandlerFunc(adminApiHostHandler(gz(httpTraceHdrs(adminAPI.RestoreConfigHistoryKVHandler)))).
				Queries("restoreId", "{restoreId:.*}")
		}

		// Config import/export bulk operations
		if enableConfigOps {
			// Get config
			adminRouter.Methods(http.MethodGet).Path(adminVersion + "/config").
				HandlerFunc(adminApiHostHandler(gz(httpTraceHdrs(adminAPI.GetConfigHandler))))
			// Set config
			adminRouter.Methods(http.MethodPut).Path(adminVersion + "/config").
				HandlerFunc(adminApiHostHandler(gz(httpTraceHdrs(adminAPI.SetConfigHandler))))
		}

		// -- IAM APIs --

		// Add policy IAM
		adminRouter.Methods(http.MethodPut).Path(adminVersion+"/add-canned-policy").
			HandlerFunc(adminApiHostHandler(gz(httpTraceAll(adminAPI.AddCannedPolicy)))).
			Queries("name", "{name:.*}")

		// Add user IAM
		adminRouter.Methods(http.MethodGet).Path(adminVersion + "/accountinfo").
			HandlerFunc(adminApiHostHandler(gz(httpTraceAll(adminAPI.AccountInfoHandler))))

		adminRouter.Methods(http.MethodPut).Path(adminVersion+"/add-user").
			HandlerFunc(adminApiHostHandler(gz(httpTraceHdrs(adminAPI.AddUser)))).
			Queries("accessKey", "{accessKey:.*}")

		adminRouter.Methods(http.MethodPut).Path(adminVersion+"/set-user-status").
			HandlerFunc(adminApiHostHandler(gz(httpTraceHdrs(adminAPI.SetUserStatus)))).
			Queries("accessKey", "{accessKey:.*}").Queries("status", "{status:.*}")

		// Service accounts ops
		adminRouter.Methods(http.MethodPut).Path(adminVersion + "/add-service-account").
			HandlerFunc(adminApiHostHandler(gz(httpTraceHdrs(adminAPI.AddServiceAccount))))
		adminRouter.Methods(http.MethodPost).Path(adminVersion+"/update-service-account").
			HandlerFunc(adminApiHostHandler(gz(httpTraceHdrs(adminAPI.UpdateServiceAccount)))).
			Queries("accessKey", "{accessKey:.*}")
		adminRouter.Methods(http.MethodGet).Path(adminVersion+"/info-service-account").
			HandlerFunc(adminApiHostHandler(gz(httpTraceHdrs(adminAPI.InfoServiceAccount)))).
			Queries("accessKey", "{accessKey:.*}")
		adminRouter.Methods(http.MethodGet).Path(adminVersion + "/list-service-accounts").
			HandlerFunc(adminApiHostHandler(gz(httpTraceHdrs(adminAPI.ListServiceAccounts))))
		adminRouter.Methods(http.MethodDelete).Path(adminVersion+"/delete-service-account").
			HandlerFunc(adminApiHostHandler(gz(httpTraceHdrs(adminAPI.DeleteServiceAccount)))).
			Queries("accessKey", "{accessKey:.*}")

		// Info policy IAM latest
		adminRouter.Methods(http.MethodGet).Path(adminVersion+"/info-canned-policy").
			HandlerFunc(adminApiHostHandler(gz(httpTraceHdrs(adminAPI.InfoCannedPolicy)))).
			Queries("name", "{name:.*}")
		// List policies latest
		adminRouter.Methods(http.MethodGet).Path(adminVersion+"/list-canned-policies").
			HandlerFunc(adminApiHostHandler(gz(httpTraceHdrs(adminAPI.ListBucketPolicies)))).
			Queries("bucket", "{bucket:.*}")
		adminRouter.Methods(http.MethodGet).Path(adminVersion + "/list-canned-policies").
			HandlerFunc(adminApiHostHandler(gz(httpTraceHdrs(adminAPI.ListCannedPolicies))))

		// Remove policy IAM
		adminRouter.Methods(http.MethodDelete).Path(adminVersion+"/remove-canned-policy").
			HandlerFunc(adminApiHostHandler(gz(httpTraceHdrs(adminAPI.RemoveCannedPolicy)))).
			Queries("name", "{name:.*}")

		// Set user or group policy
		adminRouter.Methods(http.MethodPut).Path(adminVersion+"/set-user-or-group-policy").
			HandlerFunc(adminApiHostHandler(gz(httpTraceHdrs(adminAPI.SetPolicyForUserOrGroup)))).
			Queries("policyName", "{policyName:.*}", "userOrGroup", "{userOrGroup:.*}", "isGroup", "{isGroup:true|false}")

		// Remove user IAM
		adminRouter.Methods(http.MethodDelete).Path(adminVersion+"/remove-user").
			HandlerFunc(adminApiHostHandler(gz(httpTraceHdrs(adminAPI.RemoveUser)))).
			Queries("accessKey", "{accessKey:.*}")

		// List users
		adminRouter.Methods(http.MethodGet).Path(adminVersion+"/list-users").
			HandlerFunc(adminApiHostHandler(gz(httpTraceHdrs(adminAPI.ListBucketUsers)))).
			Queries("bucket", "{bucket:.*}")
		adminRouter.Methods(http.MethodGet).Path(adminVersion + "/list-users").
			HandlerFunc(adminApiHostHandler(gz(httpTraceHdrs(adminAPI.ListUsers))))

		// User info
		adminRouter.Methods(http.MethodGet).Path(adminVersion+"/user-info").
			HandlerFunc(adminApiHostHandler(gz(httpTraceHdrs(adminAPI.GetUserInfo)))).
			Queries("accessKey", "{accessKey:.*}")
		// Add/Remove members from group
		adminRouter.Methods(http.MethodPut).Path(adminVersion + "/update-group-members").
			HandlerFunc(adminApiHostHandler(gz(httpTraceHdrs(adminAPI.UpdateGroupMembers))))

		// Get Group
		adminRouter.Methods(http.MethodGet).Path(adminVersion+"/group").
			HandlerFunc(adminApiHostHandler(gz(httpTraceHdrs(adminAPI.GetGroup)))).
			Queries("group", "{group:.*}")

		// List Groups
		adminRouter.Methods(http.MethodGet).Path(adminVersion + "/groups").
			HandlerFunc(adminApiHostHandler(gz(httpTraceHdrs(adminAPI.ListGroups))))

		// Set Group Status
		adminRouter.Methods(http.MethodPut).Path(adminVersion+"/set-group-status").
			HandlerFunc(adminApiHostHandler(gz(httpTraceHdrs(adminAPI.SetGroupStatus)))).
			Queries("group", "{group:.*}").Queries("status", "{status:.*}")

		// Export IAM info to zipped file
		adminRouter.Methods(http.MethodGet).Path(adminVersion + "/export-iam").
			HandlerFunc(adminApiHostHandler(httpTraceHdrs(adminAPI.ExportIAM)))

		// Import IAM info
		adminRouter.Methods(http.MethodPut).Path(adminVersion + "/import-iam").
			HandlerFunc(adminApiHostHandler(httpTraceHdrs(adminAPI.ImportIAM)))

		// IDentity Provider configuration APIs
		adminRouter.Methods(http.MethodPut).Path(adminVersion+"/idp-config").
			HandlerFunc(adminApiHostHandler(gz(httpTraceHdrs(adminAPI.SetIdentityProviderCfg)))).
			Queries("type", "{type:.*}").Queries("name", "{name:.*}")
		adminRouter.Methods(http.MethodGet).Path(adminVersion+"/idp-config").
			HandlerFunc(adminApiHostHandler(gz(httpTraceHdrs(adminAPI.GetIdentityProviderCfg)))).
			Queries("type", "{type:.*}")
		adminRouter.Methods(http.MethodDelete).Path(adminVersion+"/idp-config").
			HandlerFunc(adminApiHostHandler(gz(httpTraceHdrs(adminAPI.DeleteIdentityProviderCfg)))).
			Queries("type", "{type:.*}").Queries("name", "{name:.*}")

		// -- END IAM APIs --

		// GetBucketQuotaConfig
		adminRouter.Methods(http.MethodGet).Path(adminVersion+"/get-bucket-quota").
			HandlerFunc(adminApiHostHandler(gz(httpTraceHdrs(adminAPI.GetBucketQuotaConfigHandler)))).
			Queries("bucket", "{bucket:.*}")
		// PutBucketQuotaConfig
		adminRouter.Methods(http.MethodPut).Path(adminVersion+"/set-bucket-quota").
			HandlerFunc(adminApiHostHandler(gz(httpTraceHdrs(adminAPI.PutBucketQuotaConfigHandler)))).
			Queries("bucket", "{bucket:.*}")

		// Bucket replication operations
		// GetBucketTargetHandler
		adminRouter.Methods(http.MethodGet).Path(adminVersion+"/list-remote-targets").HandlerFunc(
			adminApiHostHandler(gz(httpTraceHdrs(adminAPI.ListRemoteTargetsHandler)))).
			Queries("bucket", "{bucket:.*}", "type", "{type:.*}")
		// SetRemoteTargetHandler
		adminRouter.Methods(http.MethodPut).Path(adminVersion+"/set-remote-target").HandlerFunc(
			adminApiHostHandler(gz(httpTraceHdrs(adminAPI.SetRemoteTargetHandler)))).
			Queries("bucket", "{bucket:.*}")
		// RemoveRemoteTargetHandler
		adminRouter.Methods(http.MethodDelete).Path(adminVersion+"/remove-remote-target").HandlerFunc(
			adminApiHostHandler(gz(httpTraceHdrs(adminAPI.RemoveRemoteTargetHandler)))).
			Queries("bucket", "{bucket:.*}", "arn", "{arn:.*}")
		// ReplicationDiff - MinIO extension API
		adminRouter.Methods(http.MethodPost).Path(adminVersion+"/replication/diff").HandlerFunc(
			adminApiHostHandler(gz(httpTraceHdrs(adminAPI.ReplicationDiffHandler)))).
			Queries("bucket", "{bucket:.*}")

		// Batch job operations
		adminRouter.Methods(http.MethodPost).Path(adminVersion + "/start-job").HandlerFunc(
			adminApiHostHandler(gz(httpTraceHdrs(adminAPI.StartBatchJob))))

		adminRouter.Methods(http.MethodGet).Path(adminVersion + "/list-jobs").HandlerFunc(
			adminApiHostHandler(gz(httpTraceHdrs(adminAPI.ListBatchJobs))))

		adminRouter.Methods(http.MethodGet).Path(adminVersion + "/describe-job").HandlerFunc(
			adminApiHostHandler(gz(httpTraceHdrs(adminAPI.DescribeBatchJob))))

		// Bucket migration operations
		// ExportBucketMetaHandler
		adminRouter.Methods(http.MethodGet).Path(adminVersion + "/export-bucket-metadata").HandlerFunc(
			adminApiHostHandler(gz(httpTraceHdrs(adminAPI.ExportBucketMetadataHandler))))
		// ImportBucketMetaHandler
		adminRouter.Methods(http.MethodPut).Path(adminVersion + "/import-bucket-metadata").HandlerFunc(
			adminApiHostHandler(gz(httpTraceHdrs(adminAPI.ImportBucketMetadataHandler))))

		// Remote Tier management operations
		adminRouter.Methods(http.MethodPut).Path(adminVersion + "/tier").
			HandlerFunc(adminApiHostHandler(gz(httpTraceHdrs(adminAPI.AddTierHandler))))
		adminRouter.Methods(http.MethodPost).Path(adminVersion + "/tier/{tier}").
			HandlerFunc(adminApiHostHandler(gz(httpTraceHdrs(adminAPI.EditTierHandler))))
		adminRouter.Methods(http.MethodGet).Path(adminVersion + "/tier").
			HandlerFunc(adminApiHostHandler(gz(httpTraceHdrs(adminAPI.ListTierHandler))))
		adminRouter.Methods(http.MethodDelete).Path(adminVersion + "/tier/{tier}").
			HandlerFunc(adminApiHostHandler(gz(httpTraceHdrs(adminAPI.RemoveTierHandler))))
		adminRouter.Methods(http.MethodGet).Path(adminVersion + "/tier/{tier}").
			HandlerFunc(adminApiHostHandler(gz(httpTraceHdrs(adminAPI.VerifyTierHandler))))
		// Tier stats
		adminRouter.Methods(http.MethodGet).Path(adminVersion + "/tier-stats").
			HandlerFunc(adminApiHostHandler(gz(httpTraceHdrs(adminAPI.TierStatsHandler))))

		// Cluster Replication APIs
		adminRouter.Methods(http.MethodPut).Path(adminVersion + "/site-replication/add").
			HandlerFunc(adminApiHostHandler(gz(httpTraceHdrs(adminAPI.SiteReplicationAdd))))
		adminRouter.Methods(http.MethodPut).Path(adminVersion + "/site-replication/remove").
			HandlerFunc(adminApiHostHandler(gz(httpTraceHdrs(adminAPI.SiteReplicationRemove))))
		adminRouter.Methods(http.MethodGet).Path(adminVersion + "/site-replication/info").
			HandlerFunc(adminApiHostHandler(gz(httpTraceHdrs(adminAPI.SiteReplicationInfo))))
		adminRouter.Methods(http.MethodGet).Path(adminVersion + "/site-replication/metainfo").
			HandlerFunc(adminApiHostHandler(gz(httpTraceHdrs(adminAPI.SiteReplicationMetaInfo))))
		adminRouter.Methods(http.MethodGet).Path(adminVersion + "/site-replication/status").
			HandlerFunc(adminApiHostHandler(gz(httpTraceHdrs(adminAPI.SiteReplicationStatus))))

		adminRouter.Methods(http.MethodPut).Path(adminVersion + "/site-replication/peer/join").
			HandlerFunc(adminApiHostHandler(gz(httpTraceHdrs(adminAPI.SRPeerJoin))))
		adminRouter.Methods(http.MethodPut).Path(adminVersion+"/site-replication/peer/bucket-ops").
			HandlerFunc(adminApiHostHandler(gz(httpTraceHdrs(adminAPI.SRPeerBucketOps)))).
			Queries("bucket", "{bucket:.*}").Queries("operation", "{operation:.*}")
		adminRouter.Methods(http.MethodPut).Path(adminVersion + "/site-replication/peer/iam-item").
			HandlerFunc(adminApiHostHandler(gz(httpTraceHdrs(adminAPI.SRPeerReplicateIAMItem))))
		adminRouter.Methods(http.MethodPut).Path(adminVersion + "/site-replication/peer/bucket-meta").
			HandlerFunc(adminApiHostHandler(gz(httpTraceHdrs(adminAPI.SRPeerReplicateBucketItem))))
		adminRouter.Methods(http.MethodGet).Path(adminVersion + "/site-replication/peer/idp-settings").
			HandlerFunc(adminApiHostHandler(gz(httpTraceHdrs(adminAPI.SRPeerGetIDPSettings))))
		adminRouter.Methods(http.MethodPut).Path(adminVersion + "/site-replication/edit").
			HandlerFunc(adminApiHostHandler(gz(httpTraceHdrs(adminAPI.SiteReplicationEdit))))
		adminRouter.Methods(http.MethodPut).Path(adminVersion + "/site-replication/peer/edit").
			HandlerFunc(adminApiHostHandler(gz(httpTraceHdrs(adminAPI.SRPeerEdit))))
		adminRouter.Methods(http.MethodPut).Path(adminVersion + "/site-replication/peer/remove").
			HandlerFunc(adminApiHostHandler(gz(httpTraceHdrs(adminAPI.SRPeerRemove))))

		if globalIsDistErasure {
			// Top locks
			adminRouter.Methods(http.MethodGet).Path(adminVersion + "/top/locks").
				HandlerFunc(adminApiHostHandler(gz(httpTraceHdrs(adminAPI.TopLocksHandler))))
			// Force unlocks paths
			adminRouter.Methods(http.MethodPost).Path(adminVersion+"/force-unlock").
				Queries("paths", "{paths:.*}").
				HandlerFunc(adminApiHostHandler(gz(httpTraceHdrs(adminAPI.ForceUnlockHandler))))
		}

		adminRouter.Methods(http.MethodPost).Path(adminVersion + "/speedtest").
			HandlerFunc(adminApiHostHandler(httpTraceHdrs(adminAPI.SpeedTestHandler)))
		adminRouter.Methods(http.MethodPost).Path(adminVersion + "/speedtest/object").
			HandlerFunc(adminApiHostHandler(httpTraceHdrs(adminAPI.ObjectSpeedTestHandler)))
		adminRouter.Methods(http.MethodPost).Path(adminVersion + "/speedtest/drive").
			HandlerFunc(adminApiHostHandler(httpTraceHdrs(adminAPI.DriveSpeedtestHandler)))
		adminRouter.Methods(http.MethodPost).Path(adminVersion + "/speedtest/net").
			HandlerFunc(adminApiHostHandler(httpTraceHdrs(adminAPI.NetperfHandler)))

		// HTTP Trace
		adminRouter.Methods(http.MethodGet).Path(adminVersion + "/trace").
			HandlerFunc(adminApiHostHandler(gz(http.HandlerFunc(adminAPI.TraceHandler))))

		// Console Logs
		adminRouter.Methods(http.MethodGet).Path(adminVersion + "/log").
			HandlerFunc(adminApiHostHandler(gz(httpTraceAll(adminAPI.ConsoleLogHandler))))

		// -- KMS APIs --
		//
		adminRouter.Methods(http.MethodPost).Path(adminVersion + "/kms/status").
			HandlerFunc(adminApiHostHandler(gz(httpTraceAll(adminAPI.KMSStatusHandler))))
		adminRouter.Methods(http.MethodPost).Path(adminVersion+"/kms/key/create").
			HandlerFunc(adminApiHostHandler(gz(httpTraceAll(adminAPI.KMSCreateKeyHandler)))).
			Queries("key-id", "{key-id:.*}")
		adminRouter.Methods(http.MethodGet).Path(adminVersion + "/kms/key/status").
			HandlerFunc(adminApiHostHandler(gz(httpTraceAll(adminAPI.KMSKeyStatusHandler))))

		if !globalIsGateway {
			// Keep obdinfo for backward compatibility with mc
			adminRouter.Methods(http.MethodGet).Path(adminVersion + "/obdinfo").
				HandlerFunc(adminApiHostHandler(gz(httpTraceHdrs(adminAPI.HealthInfoHandler))))
			// -- Health API --
			adminRouter.Methods(http.MethodGet).Path(adminVersion + "/healthinfo").
				HandlerFunc(adminApiHostHandler(gz(httpTraceHdrs(adminAPI.HealthInfoHandler))))
			adminRouter.Methods(http.MethodGet).Path(adminVersion + "/bandwidth").
				HandlerFunc(adminApiHostHandler(gz(httpTraceHdrs(adminAPI.BandwidthMonitorHandler))))
		}
	}

	// If none of the routes match add default error handler routes
	adminRouter.NotFoundHandler = httpTraceAll(errorResponseHandler)
	adminRouter.MethodNotAllowedHandler = httpTraceAll(methodNotAllowedHandler("Admin"))
}
