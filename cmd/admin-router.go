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

package cmd

import router "github.com/gorilla/mux"

// adminAPIHandlers implements and provides http handlers for Minio admin API.
type adminAPIHandlers struct {
}

func registerAdminRouter(mux *router.Router) {

	adminAPI := adminAPIHandlers{}
	// Admin router
	adminRouter := mux.NewRoute().PathPrefix("/").Subrouter()
	/// Admin operations
	// Service status
	adminRouter.Methods("GET").Headers("service", "").HandlerFunc(adminAPI.ServiceStatusHandler)
	// Service stop
	adminRouter.Methods("POST").Headers("service", "", minioAdminOpHeader, "stop").HandlerFunc(adminAPI.ServiceStopHandler)
	// Service restart
	adminRouter.Methods("POST").Headers("service", "", minioAdminOpHeader, "restart").HandlerFunc(adminAPI.ServiceRestartHandler)
}
