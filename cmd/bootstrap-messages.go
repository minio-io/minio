// Copyright (c) 2015-2023 MinIO, Inc.
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
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/minio/madmin-go/v2"
	"github.com/minio/minio/internal/pubsub"
)

const bootstrapMsgsLimit = 4 << 10

type bootstrapInfo struct {
	msg    string
	ts     time.Time
	source string
}
type bootstrapTracer struct {
	mu         sync.RWMutex
	idx        int
	info       [bootstrapMsgsLimit]bootstrapInfo
	lastUpdate time.Time
}

var globalBootstrapTracer = &bootstrapTracer{}

func (bs *bootstrapTracer) DropEvents() {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	if time.Now().UTC().Sub(bs.lastUpdate) > 24*time.Hour {
		bs.info = [4096]bootstrapInfo{}
		bs.idx = 0
	}
}

func (bs *bootstrapTracer) Empty() bool {
	var empty bool
	bs.mu.RLock()
	empty = bs.info[0].msg == ""
	bs.mu.RUnlock()

	return empty
}

func (bs *bootstrapTracer) Record(msg string) {
	source := getSource(2)
	bs.mu.Lock()
	now := time.Now().UTC()
	bs.info[bs.idx] = bootstrapInfo{
		msg:    msg,
		ts:     now,
		source: source,
	}
	bs.lastUpdate = now
	bs.idx++
	if bs.idx >= bootstrapMsgsLimit {
		bs.idx = 0 // circular buffer
	}
	bs.mu.Unlock()
}

func (bs *bootstrapTracer) Events() []madmin.TraceInfo {
	var info [bootstrapMsgsLimit]bootstrapInfo
	var idx int

	bs.mu.RLock()
	idx = bs.idx
	tail := bootstrapMsgsLimit - idx
	copy(info[tail:], bs.info[:idx])
	copy(info[:tail], bs.info[idx:])
	bs.mu.RUnlock()

	traceInfo := make([]madmin.TraceInfo, 0, bootstrapMsgsLimit)
	for i := 0; i < bootstrapMsgsLimit; i++ {
		if info[i].ts.IsZero() {
			continue // skip empty events
		}
		traceInfo = append(traceInfo, madmin.TraceInfo{
			TraceType: madmin.TraceBootstrap,
			Time:      info[i].ts,
			NodeName:  globalLocalNodeName,
			FuncName:  "BOOTSTRAP",
			Message:   fmt.Sprintf("%s %s", info[i].source, info[i].msg),
		})
	}
	return traceInfo
}

func (bs *bootstrapTracer) Publish(ctx context.Context, trace *pubsub.PubSub[madmin.TraceInfo, madmin.TraceType]) {
	if bs.Empty() {
		return
	}
	for _, bsEvent := range bs.Events() {
		select {
		case <-ctx.Done():
		default:
			trace.Publish(bsEvent)
		}
	}
}
