// +build openbsd

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

package disk

import "syscall"

// getFSType returns the filesystem type of the underlying mounted filesystem
func getFSType(path string) (string, error) {
	s := syscall.Statfs_t{}
	err := syscall.Statfs(path, &s)
	if err != nil {
		return "", err
	}
	// F_fstypename's type is []int8
	fsTypeBytes := []byte{}
	for _, i := range s.F_fstypename {
		fsTypeBytes = append(fsTypeBytes, byte(i))
	}
	return string(fsTypeBytes), nil
}
