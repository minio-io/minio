/*
 * Minio Cloud Storage, (C) 2017 Minio, Inc.
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

import (
	"fmt"
	"net"
	"net/url"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/minio/minio-go/pkg/set"
)

// EndpointType - enum for endpoint type.
type EndpointType int

const (
	// PathEndpointType - path style endpoint type enum.
	PathEndpointType EndpointType = iota + 1

	// URLEndpointType - URL style endpoint type enum.
	URLEndpointType
)

// Endpoint - any type of endpoint.
type Endpoint struct {
	*url.URL
	IsLocal bool
}

func (endpoint Endpoint) String() string {
	if endpoint.Host == "" {
		return endpoint.Path
	}

	return endpoint.URL.String()
}

// Type - returns type of endpoint.
func (endpoint Endpoint) Type() EndpointType {
	if endpoint.Host == "" {
		return PathEndpointType
	}

	return URLEndpointType
}

// SetHTTPS - sets secure http for URLEndpointType.
func (endpoint Endpoint) SetHTTPS() {
	if endpoint.Host != "" {
		endpoint.Scheme = "https"
	}
}

// SetHTTP - sets insecure http for URLEndpointType.
func (endpoint Endpoint) SetHTTP() {
	if endpoint.Host != "" {
		endpoint.Scheme = "http"
	}
}

// NewEndpoint - returns new endpoint based on given arguments.
func NewEndpoint(arg string) (Endpoint, error) {
	// isEmptyPath - check whether given path is not empty.
	isEmptyPath := func(path string) bool {
		return path == "" || path == "." || path == "/" || path == `\`
	}

	if isEmptyPath(arg) {
		return Endpoint{}, fmt.Errorf("empty or root endpoint is not supported")
	}

	var isLocal bool
	u, err := url.Parse(arg)
	if err == nil && u.Host != "" {
		// URL style of endpoint.
		// Valid URL style endpoint is
		// - Scheme field must contain "http" or "https"
		// - All field should be empty except Host and Path.
		if !((u.Scheme == "http" || u.Scheme == "https") &&
			u.User == nil && u.Opaque == "" && u.ForceQuery == false && u.RawQuery == "" && u.Fragment == "") {
			return Endpoint{}, fmt.Errorf("invalid URL endpoint format")
		}

		host, port, err := net.SplitHostPort(u.Host)
		if err != nil {
			if !strings.Contains(err.Error(), "missing port in address") {
				return Endpoint{}, fmt.Errorf("invalid URL endpoint format: %s", err)
			}

			host = u.Host
		} else {
			var p int
			p, err = strconv.Atoi(port)
			if err != nil {
				return Endpoint{}, fmt.Errorf("invalid URL endpoint format: invalid port number")
			} else if p < 1 || p > 65535 {
				return Endpoint{}, fmt.Errorf("invalid URL endpoint format: port number must be between 1 to 65535")
			}
		}

		if host == "" {
			return Endpoint{}, fmt.Errorf("invalid URL endpoint format: empty host name")
		}

		// As this is path in the URL, we should use path package, not filepath package.
		// On MS Windows, filepath.Clean() converts into Windows path style ie `/foo` becomes `\foo`
		u.Path = path.Clean(u.Path)
		if isEmptyPath(u.Path) {
			return Endpoint{}, fmt.Errorf("empty or root path is not supported in URL endpoint")
		}

		// Get IPv4 address of the host.
		hostIPs, err := getHostIP4(host)
		if err != nil {
			return Endpoint{}, err
		}

		// If intersection of two IP sets is not empty, then the host is local host.
		isLocal = !localIP4.Intersection(hostIPs).IsEmpty()
	} else {
		u = &url.URL{Path: path.Clean(arg)}
		isLocal = true
	}

	return Endpoint{
		URL:     u,
		IsLocal: isLocal,
	}, nil
}

// EndpointList - list of same type of endpoint.
type EndpointList []Endpoint

// Swap - helper method for sorting.
func (endpoints EndpointList) Swap(i, j int) {
	endpoints[i], endpoints[j] = endpoints[j], endpoints[i]
}

// Len - helper method for sorting.
func (endpoints EndpointList) Len() int {
	return len(endpoints)
}

// Less - helper method for sorting.
func (endpoints EndpointList) Less(i, j int) bool {
	return endpoints[i].String() < endpoints[j].String()
}

// SetHTTPS - sets secure http for URLEndpointType.
func (endpoints EndpointList) SetHTTPS() {
	for i := range endpoints {
		endpoints[i].SetHTTPS()
	}
}

// SetHTTP - sets insecure http for URLEndpointType.
func (endpoints EndpointList) SetHTTP() {
	for i := range endpoints {
		endpoints[i].SetHTTP()
	}
}

// NewEndpointList - returns new endpoint list based on input args.
func NewEndpointList(args ...string) (endpoints EndpointList, err error) {
	// isValidDistribution - checks whether given count is a valid distribution for erasure coding.
	isValidDistribution := func(count int) bool {
		return (count >= 4 && count <= 16 && count%2 == 0)
	}

	// Check whether no. of args are valid for XL distribution.
	if !isValidDistribution(len(args)) {
		return nil, fmt.Errorf("total endpoints %d found. For XL/Distribute, it should be 4, 6, 8, 10, 12, 14 or 16", len(args))
	}

	var endpointType EndpointType
	var scheme string

	uniqueArgs := set.NewStringSet()
	// Loop through args and adds to endpoint list.
	for i, arg := range args {
		endpoint, err := NewEndpoint(arg)
		if err != nil {
			return nil, fmt.Errorf("'%s': %s", arg, err.Error())
		}

		// All endpoints have to be same type and scheme if applicable.
		if i == 0 {
			endpointType = endpoint.Type()
			scheme = endpoint.Scheme
		} else if endpoint.Type() != endpointType {
			return nil, fmt.Errorf("mixed style endpoints are not supported")
		} else if endpoint.Scheme != scheme {
			return nil, fmt.Errorf("mixed scheme is not supported")
		}

		arg = endpoint.String()
		if uniqueArgs.Contains(arg) {
			return nil, fmt.Errorf("duplicate endpoints found")
		}
		uniqueArgs.Add(arg)

		endpoints = append(endpoints, endpoint)
	}

	sort.Sort(endpoints)

	return endpoints, nil
}

// CreateEndpoints - validates and creates new endpoints for given args.
func CreateEndpoints(serverAddr string, args ...string) (string, EndpointList, SetupType, error) {
	var endpoints EndpointList
	var setupType SetupType
	var err error

	// Check whether serverAddr is valid for this host.
	if err = CheckLocalServerAddr(serverAddr); err != nil {
		return serverAddr, endpoints, setupType, err
	}

	_, serverAddrPort := mustSplitHostPort(serverAddr)

	// For single arg, return FS setup.
	if len(args) == 1 {
		var endpoint Endpoint
		endpoint, err = NewEndpoint(args[0])
		if err != nil {
			return serverAddr, endpoints, setupType, err
		}

		if endpoint.Type() != PathEndpointType {
			return serverAddr, endpoints, setupType, fmt.Errorf("use path style endpoint for FS setup")
		}

		endpoints = append(endpoints, endpoint)
		setupType = FSSetupType
		return serverAddr, endpoints, setupType, nil
	}

	// Convert args to endpoints
	if endpoints, err = NewEndpointList(args...); err != nil {
		return serverAddr, endpoints, setupType, err
	}

	// Return XL setup when all endpoints are path style.
	if endpoints[0].Type() == PathEndpointType {
		setupType = XLSetupType
		return serverAddr, endpoints, setupType, nil
	}

	// Here all endpoints are URL style.
	endpointPathSet := set.NewStringSet()
	localEndpointCount := 0
	localServerAddrSet := set.NewStringSet()
	localPortSet := set.NewStringSet()
	for _, endpoint := range endpoints {
		endpointPathSet.Add(endpoint.Path)
		if endpoint.IsLocal {
			localServerAddrSet.Add(endpoint.Host)

			var port string
			_, port, err = net.SplitHostPort(endpoint.Host)
			if err != nil {
				port = serverAddrPort
			}

			localPortSet.Add(port)

			localEndpointCount++
		}
	}

	// No local endpoint found.
	if localEndpointCount == 0 {
		return serverAddr, endpoints, setupType, fmt.Errorf("no endpoint found for this host")
	}

	// Check whether same path is not used in endpoints of a host.
	{
		pathIPMap := make(map[string]set.StringSet)
		for _, endpoint := range endpoints {
			var host string
			host, _, err = net.SplitHostPort(endpoint.Host)
			if err != nil {
				host = endpoint.Host
			}
			hostIPSet, _ := getHostIP4(host)
			if IPSet, ok := pathIPMap[endpoint.Path]; ok {
				if !IPSet.Intersection(hostIPSet).IsEmpty() {
					err = fmt.Errorf("path '%s' can not be served from different address/port", endpoint.Path)
					return serverAddr, endpoints, setupType, err
				}
			} else {
				pathIPMap[endpoint.Path] = hostIPSet
			}
		}
	}

	// Check whether serverAddrPort matches at least in one of port used in local endpoints.
	{
		if !localPortSet.Contains(serverAddrPort) {
			if len(localPortSet) > 1 {
				err = fmt.Errorf("port number in server address must match with one of the port in local endpoints")
			} else {
				err = fmt.Errorf("server address and local endpoint have different ports")
			}

			return serverAddr, endpoints, setupType, err
		}
	}

	// If all endpoints are pointing to local host and having same port number, then this is XL setup using URL style endpoints.
	if len(endpoints) == localEndpointCount && len(localPortSet) == 1 {
		if len(localServerAddrSet) > 1 {
			// TODO: Eventhough all endpoints are local, the local host is referred by different IP/name.
			// eg '172.0.0.1', 'localhost' and 'mylocalhostname' point to same local host.
			//
			// In this case, we bind to 0.0.0.0 ie to all interfaces.
			// The actual way to do is bind to only IPs in uniqueLocalHosts.
			serverAddr = net.JoinHostPort("", serverAddrPort)
		}

		endpointPaths := endpointPathSet.ToSlice()
		endpoints, _ = NewEndpointList(endpointPaths...)
		setupType = XLSetupType
		return serverAddr, endpoints, setupType, nil
	}

	// Add missing port in all endpoints.
	for i := range endpoints {
		_, port, err := net.SplitHostPort(endpoints[i].Host)
		if err != nil {
			endpoints[i].Host = net.JoinHostPort(endpoints[i].Host, serverAddrPort)
		} else if endpoints[i].IsLocal && serverAddrPort != port {
			// If endpoint is local, but port is different than serverAddrPort, then make it as remote.
			endpoints[i].IsLocal = false
		}
	}

	// This is DistXL setup.
	setupType = DistXLSetupType
	return serverAddr, endpoints, setupType, nil
}

// GetRemotePeers - get hosts information other than this minio service.
func GetRemotePeers(endpoints EndpointList) []string {
	peerSet := set.NewStringSet()
	for _, endpoint := range endpoints {
		if endpoint.Type() != URLEndpointType {
			continue
		}

		peer := endpoint.Host
		if endpoint.IsLocal {
			if _, port := mustSplitHostPort(peer); port == globalMinioPort {
				continue
			}
		}

		peerSet.Add(peer)
	}

	return peerSet.ToSlice()
}
