/*
 * MinIO Cloud Storage, (C) 2017 MinIO, Inc.
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
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"reflect"
	"sync"

	"github.com/cespare/xxhash/v2"
	"github.com/klauspost/reedsolomon"
	"github.com/minio/minio/cmd/logger"
)

// Erasure - erasure encoding details.
type Erasure struct {
	encoder                  func() reedsolomon.Encoder
	dataBlocks, parityBlocks int
	blockSize                int64
}

// NewErasure creates a new ErasureStorage.
func NewErasure(ctx context.Context, dataBlocks, parityBlocks int, blockSize int64) (e Erasure, err error) {
	// Check the parameters for sanity now.
	if dataBlocks <= 0 || parityBlocks <= 0 {
		return e, reedsolomon.ErrInvShardNum
	}

	if dataBlocks+parityBlocks > 256 {
		return e, reedsolomon.ErrMaxShardNum
	}

	e = Erasure{
		dataBlocks:   dataBlocks,
		parityBlocks: parityBlocks,
		blockSize:    blockSize,
	}

	// Encoder when needed.
	var enc reedsolomon.Encoder
	var once sync.Once
	e.encoder = func() reedsolomon.Encoder {
		once.Do(func() {
			e, err := reedsolomon.New(dataBlocks, parityBlocks, reedsolomon.WithAutoGoroutines(int(e.ShardSize())))
			if err != nil {
				// Error conditions should be checked above.
				panic(err)
			}
			enc = e
		})
		return enc
	}
	return
}

// EncodeData encodes the given data and returns the erasure-coded data.
// It returns an error if the erasure coding failed.
func (e *Erasure) EncodeData(ctx context.Context, data []byte) ([][]byte, error) {
	if len(data) == 0 {
		return make([][]byte, e.dataBlocks+e.parityBlocks), nil
	}
	encoded, err := e.encoder().Split(data)
	if err != nil {
		logger.LogIf(ctx, err)
		return nil, err
	}
	if err = e.encoder().Encode(encoded); err != nil {
		logger.LogIf(ctx, err)
		return nil, err
	}
	return encoded, nil
}

// DecodeDataBlocks decodes the given erasure-coded data.
// It only decodes the data blocks but does not verify them.
// It returns an error if the decoding failed.
func (e *Erasure) DecodeDataBlocks(data [][]byte) error {
	var isZero = 0
	for _, b := range data[:] {
		if len(b) == 0 {
			isZero++
			break
		}
	}
	if isZero == 0 || isZero == len(data) {
		// If all are zero, payload is 0 bytes.
		return nil
	}
	return e.encoder().ReconstructData(data)
}

// DecodeDataAndParityBlocks decodes the given erasure-coded data and verifies it.
// It returns an error if the decoding failed.
func (e *Erasure) DecodeDataAndParityBlocks(ctx context.Context, data [][]byte) error {
	if err := e.encoder().Reconstruct(data); err != nil {
		logger.LogIf(ctx, err)
		return err
	}
	return nil
}

// ShardSize - returns actual shared size from erasure blockSize.
func (e *Erasure) ShardSize() int64 {
	return ceilFrac(e.blockSize, int64(e.dataBlocks))
}

// ShardFileSize - returns final erasure size from original size.
func (e *Erasure) ShardFileSize(totalLength int64) int64 {
	if totalLength == 0 {
		return 0
	}
	if totalLength == -1 {
		return -1
	}
	numShards := totalLength / e.blockSize
	lastBlockSize := totalLength % e.blockSize
	lastShardSize := ceilFrac(lastBlockSize, int64(e.dataBlocks))
	return numShards*e.ShardSize() + lastShardSize
}

// ShardFileOffset - returns the effective offset where erasure reading begins.
func (e *Erasure) ShardFileOffset(startOffset, length, totalLength int64) int64 {
	shardSize := e.ShardSize()
	shardFileSize := e.ShardFileSize(totalLength)
	endShard := (startOffset + length) / e.blockSize
	tillOffset := endShard*shardSize + shardSize
	if tillOffset > shardFileSize {
		tillOffset = shardFileSize
	}
	return tillOffset
}

// erasureSelfTest performs a self-test to ensure that erasure
// algorithms compute expected erasure codes. If any algorithm
// produces an incorrect value it fails with a hard error.
//
// erasureSelfTest tries to catch any issue in the erasure implementation
// early instead of silently corrupting data.
func erasureSelfTest() {
	// Approx runtime ~1ms
	var testConfigs [][2]uint8
	for total := uint8(4); total < 16; total++ {
		for data := total / 2; data < total; data++ {
			parity := total - data
			testConfigs = append(testConfigs, [2]uint8{data, parity})
		}
	}
	got := make(map[[2]uint8]map[ErasureAlgo]uint64, len(testConfigs))
	// Copied from output of fmt.Printf("%#v", got) at the end.
	want := map[[2]uint8]map[ErasureAlgo]uint64{[2]uint8{0x2, 0x2}: {0x1: 0x23fb21be2496f5d3}, [2]uint8{0x2, 0x3}: {0x1: 0xa5cd5600ba0d8e7c}, [2]uint8{0x3, 0x1}: {0x1: 0x60ab052148b010b4}, [2]uint8{0x3, 0x2}: {0x1: 0xe64927daef76435a}, [2]uint8{0x3, 0x3}: {0x1: 0x672f6f242b227b21}, [2]uint8{0x3, 0x4}: {0x1: 0x571e41ba23a6dc6}, [2]uint8{0x4, 0x1}: {0x1: 0x524eaa814d5d86e2}, [2]uint8{0x4, 0x2}: {0x1: 0x62b9552945504fef}, [2]uint8{0x4, 0x3}: {0x1: 0xcbf9065ee053e518}, [2]uint8{0x4, 0x4}: {0x1: 0x9a07581dcd03da8}, [2]uint8{0x4, 0x5}: {0x1: 0xbf2d27b55370113f}, [2]uint8{0x5, 0x1}: {0x1: 0xf71031a01d70daf}, [2]uint8{0x5, 0x2}: {0x1: 0x8e5845859939d0f4}, [2]uint8{0x5, 0x3}: {0x1: 0x7ad9161acbb4c325}, [2]uint8{0x5, 0x4}: {0x1: 0xc446b88830b4f800}, [2]uint8{0x5, 0x5}: {0x1: 0xabf1573cc6f76165}, [2]uint8{0x5, 0x6}: {0x1: 0x7b5598a85045bfb8}, [2]uint8{0x6, 0x1}: {0x1: 0xe2fc1e677cc7d872}, [2]uint8{0x6, 0x2}: {0x1: 0x7ed133de5ca6a58e}, [2]uint8{0x6, 0x3}: {0x1: 0x39ef92d0a74cc3c0}, [2]uint8{0x6, 0x4}: {0x1: 0xcfc90052bc25d20}, [2]uint8{0x6, 0x5}: {0x1: 0x71c96f6baeef9c58}, [2]uint8{0x6, 0x6}: {0x1: 0x4b79056484883e4c}, [2]uint8{0x6, 0x7}: {0x1: 0xb1a0e2427ac2dc1a}, [2]uint8{0x7, 0x1}: {0x1: 0x937ba2b7af467a22}, [2]uint8{0x7, 0x2}: {0x1: 0x5fd13a734d27d37a}, [2]uint8{0x7, 0x3}: {0x1: 0x3be2722d9b66912f}, [2]uint8{0x7, 0x4}: {0x1: 0x14c628e59011be3d}, [2]uint8{0x7, 0x5}: {0x1: 0xcc3b39ad4c083b9f}, [2]uint8{0x7, 0x6}: {0x1: 0x45af361b7de7a4ff}, [2]uint8{0x7, 0x7}: {0x1: 0x456cc320cec8a6e6}, [2]uint8{0x7, 0x8}: {0x1: 0x1867a9f4db315b5c}, [2]uint8{0x8, 0x1}: {0x1: 0xbc5756b9a9ade030}, [2]uint8{0x8, 0x2}: {0x1: 0xdfd7d9d0b3e36503}, [2]uint8{0x8, 0x3}: {0x1: 0x72bb72c2cdbcf99d}, [2]uint8{0x8, 0x4}: {0x1: 0x3ba5e9b41bf07f0}, [2]uint8{0x8, 0x5}: {0x1: 0xd7dabc15800f9d41}, [2]uint8{0x8, 0x6}: {0x1: 0xb482a6169fd270f}, [2]uint8{0x8, 0x7}: {0x1: 0x50748e0099d657e8}, [2]uint8{0x9, 0x1}: {0x1: 0xc77ae0144fcaeb6e}, [2]uint8{0x9, 0x2}: {0x1: 0x8a86c7dbebf27b68}, [2]uint8{0x9, 0x3}: {0x1: 0xa64e3be6d6fe7e92}, [2]uint8{0x9, 0x4}: {0x1: 0x239b71c41745d207}, [2]uint8{0x9, 0x5}: {0x1: 0x2d0803094c5a86ce}, [2]uint8{0x9, 0x6}: {0x1: 0xa3c2539b3af84874}, [2]uint8{0xa, 0x1}: {0x1: 0x7d30d91b89fcec21}, [2]uint8{0xa, 0x2}: {0x1: 0xfa5af9aa9f1857a3}, [2]uint8{0xa, 0x3}: {0x1: 0x84bc4bda8af81f90}, [2]uint8{0xa, 0x4}: {0x1: 0x6c1cba8631de994a}, [2]uint8{0xa, 0x5}: {0x1: 0x4383e58a086cc1ac}, [2]uint8{0xb, 0x1}: {0x1: 0x4ed2929a2df690b}, [2]uint8{0xb, 0x2}: {0x1: 0xecd6f1b1399775c0}, [2]uint8{0xb, 0x3}: {0x1: 0xc78cfbfc0dc64d01}, [2]uint8{0xb, 0x4}: {0x1: 0xb2643390973702d6}, [2]uint8{0xc, 0x1}: {0x1: 0x3b2a88686122d082}, [2]uint8{0xc, 0x2}: {0x1: 0xfd2f30a48a8e2e9}, [2]uint8{0xc, 0x3}: {0x1: 0xd5ce58368ae90b13}, [2]uint8{0xd, 0x1}: {0x1: 0x9c88e2a9d1b8fff8}, [2]uint8{0xd, 0x2}: {0x1: 0xcb8460aa4cf6613}, [2]uint8{0xe, 0x1}: {0x1: 0x78a28bbaec57996e}}
	var testData [256]byte
	for i := range testData {
		testData[i] = byte(i)
	}
	ok := true
	for algo := invalidErasureAlgo + 1; algo < lastErasureAlgo; algo++ {
		for _, conf := range testConfigs {
			failOnErr := func(err error) {
				if err != nil {
					fmt.Fprintf(os.Stderr, "%v: error on self-test [d:%d,p:%d]: %v. Unsafe to start server.\n", algo, conf[0], conf[1], err)
					os.Exit(1)
				}
			}
			e, err := NewErasure(context.Background(), int(conf[0]), int(conf[1]), blockSizeV2)
			failOnErr(err)
			encoded, err := e.EncodeData(GlobalContext, testData[:])
			failOnErr(err)
			hash := xxhash.New()
			for i, data := range encoded {
				// Write index to keep track of sizes of each.
				_, err = hash.Write([]byte{byte(i)})
				failOnErr(err)
				_, err = hash.Write(data)
				failOnErr(err)
				got[conf] = map[ErasureAlgo]uint64{algo: hash.Sum64()}
			}

			if a, b := want[conf], got[conf]; !reflect.DeepEqual(a, b) {
				fmt.Fprintf(os.Stderr, "%v: error on self-test [d:%d,p:%d]: want %#v, got %#v\n", algo, conf[0], conf[1], a, b)
				ok = false
				continue
			}
			// Delete first shard and reconstruct...
			first := encoded[0]
			encoded[0] = nil
			failOnErr(e.DecodeDataBlocks(encoded))
			if a, b := first, encoded[0]; !bytes.Equal(a, b) {
				fmt.Fprintf(os.Stderr, "%v: error on self-test [d:%d,p:%d]: want %#v, got %#v\n", algo, conf[0], conf[1], hex.EncodeToString(a), hex.EncodeToString(b))
				ok = false
				continue
			}

		}
	}
	if !ok {
		fmt.Fprint(os.Stderr, "Unsafe to start server.\n")
		os.Exit(1)
	}
}
