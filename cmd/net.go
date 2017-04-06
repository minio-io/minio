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
	"os"
	"sort"
	"strconv"
	"syscall"

	"github.com/minio/minio-go/pkg/set"
)

// IPv4 addresses of local host.
var localIP4 = mustGetLocalIP4()

// mustSplitHostPort is a wrapper to net.SplitHostPort() where error is assumed to be a fatal.
func mustSplitHostPort(hostPort string) (host, port string) {
	host, port, err := net.SplitHostPort(hostPort)
	fatalIf(err, "Unable to split host port %s", hostPort)
	return host, port
}

// mustGetLocalIP4 returns IPv4 addresses of local host.  It panics on error.
func mustGetLocalIP4() (ipList set.StringSet) {
	ipList = set.NewStringSet()
	addrs, err := net.InterfaceAddrs()
	fatalIf(err, "Unable to get IP addresses of this host.")

	for _, addr := range addrs {
		var ip net.IP
		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		}

		if ip.To4() != nil {
			ipList.Add(ip.String())
		}
	}

	return ipList
}

// getHostIP4 returns IPv4 address of given host.
func getHostIP4(host string) (ipList set.StringSet, err error) {
	ipList = set.NewStringSet()
	ips, err := net.LookupIP(host)
	if err != nil {
		return ipList, err
	}

	for _, ip := range ips {
		if ip.To4() != nil {
			ipList.Add(ip.String())
		}
	}

	return ipList, err
}

func getAPIEndpoints(serverAddr string) (apiEndpoints []string) {
	host, port := mustSplitHostPort(serverAddr)

	var ipList []string
	if host == "" {
		ipList = localIP4.ToSlice()
	} else {
		ipList = []string{host}
	}

	sort.Strings(ipList)

	scheme := httpScheme
	if globalIsSSL {
		scheme = httpsScheme
	}

	for _, ip := range ipList {
		apiEndpoints = append(apiEndpoints, fmt.Sprintf("%s://%s:%s", scheme, ip, port))
	}

	return apiEndpoints
}

// checkPortAvailability - check if given port is already in use.
// Note: The check method tries to listen on given port and closes it.
// It is possible to have a disconnected client in this tiny window of time.
func checkPortAvailability(port string) (err error) {
	// Return true if err is "address already in use" error.
	isAddrInUseErr := func(err error) (b bool) {
		if opErr, ok := err.(*net.OpError); ok {
			if sysErr, ok := opErr.Err.(*os.SyscallError); ok {
				if errno, ok := sysErr.Err.(syscall.Errno); ok {
					b = (errno == syscall.EADDRINUSE)
				}
			}
		}

		return b
	}

	network := []string{"tcp", "tcp4", "tcp6"}
	for _, n := range network {
		l, err := net.Listen(n, net.JoinHostPort("", port))
		if err == nil {
			// As we are able to listen on this network, the port is not in use.
			// Close the listener and continue check other networks.
			if err = l.Close(); err != nil {
				return err
			}
		} else if isAddrInUseErr(err) {
			// As we got EADDRINUSE error, the port is in use by other process.
			// Return the error.
			return err
		}
	}

	return nil
}

// CheckLocalServerAddr - checks if serverAddr is valid and local host.
func CheckLocalServerAddr(serverAddr string) error {
	host, port, err := net.SplitHostPort(serverAddr)
	if err != nil {
		return err
	}

	// Check whether port is a valid port number.
	p, err := strconv.Atoi(port)
	if err != nil {
		return fmt.Errorf("invalid port number")
	} else if p < 1 || p > 65536 {
		return fmt.Errorf("port number must be between 1 to 65536")
	}

	if host != "" {
		hostIPs, err := getHostIP4(host)
		if err != nil {
			return err
		}

		if localIP4.Intersection(hostIPs).IsEmpty() {
			return fmt.Errorf("host in server address should be this server")
		}
	}

	return nil
}
