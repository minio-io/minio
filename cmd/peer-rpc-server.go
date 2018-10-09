/*
 * Minio Cloud Storage, (C) 2018 Minio, Inc.
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
	"context"
	"fmt"
	"path"

	"github.com/gorilla/mux"
	"github.com/minio/minio/cmd/logger"
	xrpc "github.com/minio/minio/cmd/rpc"
	"github.com/minio/minio/pkg/auth"
	"github.com/minio/minio/pkg/event"
	xnet "github.com/minio/minio/pkg/net"
	"github.com/minio/minio/pkg/policy"
)

const peerServiceName = "Peer"
const peerServiceSubPath = "/s3/remote"

var peerServicePath = path.Join(minioReservedBucketPath, peerServiceSubPath)

// peerRPCReceiver - Peer RPC receiver for peer RPC server.
type peerRPCReceiver struct{}

// DeleteBucketArgs - delete bucket RPC arguments.
type DeleteBucketArgs struct {
	AuthArgs
	BucketName string
}

// DeleteBucket - handles delete bucket RPC call which removes all values of given bucket in global NotificationSys object.
func (receiver *peerRPCReceiver) DeleteBucket(args *DeleteBucketArgs, reply *VoidReply) error {
	globalNotificationSys.RemoveNotification(args.BucketName)
	globalPolicySys.Remove(args.BucketName)
	globalVersioningSys.Remove(args.BucketName)
	return nil
}

// SetBucketVersioningArgs - set bucket versioning RPC arguments.
type SetBucketVersioningArgs struct {
	AuthArgs
	BucketName string
	Versioning VersioningConfiguration
}

// SetBucketVersioning - handles set bucket versioning RPC call which sets global versioning status for the bucket
func (receiver *peerRPCReceiver) SetBucketVersioning(args *SetBucketVersioningArgs, reply *VoidReply) error {
	globalVersioningSys.Set(args.BucketName, args.Versioning)
	return nil
}

// SetBucketPolicyArgs - set bucket policy RPC arguments.
type SetBucketPolicyArgs struct {
	AuthArgs
	BucketName string
	Policy     policy.Policy
}

// SetBucketPolicy - handles set bucket policy RPC call which adds bucket policy to globalPolicySys.
func (receiver *peerRPCReceiver) SetBucketPolicy(args *SetBucketPolicyArgs, reply *VoidReply) error {
	globalPolicySys.Set(args.BucketName, args.Policy)
	return nil
}

// RemoveBucketPolicyArgs - delete bucket policy RPC arguments.
type RemoveBucketPolicyArgs struct {
	AuthArgs
	BucketName string
}

// RemoveBucketPolicy - handles delete bucket policy RPC call which removes bucket policy to globalPolicySys.
func (receiver *peerRPCReceiver) RemoveBucketPolicy(args *RemoveBucketPolicyArgs, reply *VoidReply) error {
	globalPolicySys.Remove(args.BucketName)
	return nil
}

// PutBucketNotificationArgs - put bucket notification RPC arguments.
type PutBucketNotificationArgs struct {
	AuthArgs
	BucketName string
	RulesMap   event.RulesMap
}

// PutBucketNotification - handles put bucket notification RPC call which adds rules to given bucket to global NotificationSys object.
func (receiver *peerRPCReceiver) PutBucketNotification(args *PutBucketNotificationArgs, reply *VoidReply) error {
	globalNotificationSys.AddRulesMap(args.BucketName, args.RulesMap)
	return nil
}

// ListenBucketNotificationArgs - listen bucket notification RPC arguments.
type ListenBucketNotificationArgs struct {
	AuthArgs   `json:"-"`
	BucketName string         `json:"-"`
	EventNames []event.Name   `json:"eventNames"`
	Pattern    string         `json:"pattern"`
	TargetID   event.TargetID `json:"targetId"`
	Addr       xnet.Host      `json:"addr"`
}

// ListenBucketNotification - handles listen bucket notification RPC call. It creates PeerRPCClient target which pushes requested events to target in remote peer.
func (receiver *peerRPCReceiver) ListenBucketNotification(args *ListenBucketNotificationArgs, reply *VoidReply) error {
	rpcClient := globalNotificationSys.GetPeerRPCClient(args.Addr)
	if rpcClient == nil {
		return fmt.Errorf("unable to find PeerRPCClient for provided address %v. This happens only if remote and this minio run with different set of endpoints", args.Addr)
	}

	target := NewPeerRPCClientTarget(args.BucketName, args.TargetID, rpcClient)
	rulesMap := event.NewRulesMap(args.EventNames, args.Pattern, target.ID())
	if err := globalNotificationSys.AddRemoteTarget(args.BucketName, target, rulesMap); err != nil {
		reqInfo := &logger.ReqInfo{BucketName: target.bucketName}
		reqInfo.AppendTags("target", target.id.Name)
		ctx := logger.SetReqInfo(context.Background(), reqInfo)
		logger.LogIf(ctx, err)
		return err
	}
	return nil
}

// RemoteTargetExistArgs - remote target ID exist RPC arguments.
type RemoteTargetExistArgs struct {
	AuthArgs
	BucketName string
	TargetID   event.TargetID
}

// RemoteTargetExist - handles target ID exist RPC call which checks whether given target ID is a HTTP client target or not.
func (receiver *peerRPCReceiver) RemoteTargetExist(args *RemoteTargetExistArgs, reply *bool) error {
	*reply = globalNotificationSys.RemoteTargetExist(args.BucketName, args.TargetID)
	return nil
}

// SendEventArgs - send event RPC arguments.
type SendEventArgs struct {
	AuthArgs
	Event      event.Event
	TargetID   event.TargetID
	BucketName string
}

// SendEvent - handles send event RPC call which sends given event to target by given target ID.
func (receiver *peerRPCReceiver) SendEvent(args *SendEventArgs, reply *bool) error {
	// Set default to true to keep the target.
	*reply = true
	errs := globalNotificationSys.send(args.BucketName, args.Event, args.TargetID)

	for i := range errs {
		reqInfo := (&logger.ReqInfo{}).AppendTags("Event", args.Event.EventName.String())
		reqInfo.AppendTags("targetName", args.TargetID.Name)
		ctx := logger.SetReqInfo(context.Background(), reqInfo)
		logger.LogIf(ctx, errs[i].Err)

		*reply = false // send failed i.e. do not keep the target.
		return errs[i].Err
	}

	return nil
}

// SetCredentialsArgs - set credentials RPC arguments.
type SetCredentialsArgs struct {
	AuthArgs
	Credentials auth.Credentials
}

// SetCredentials - handles set credentials RPC call.
func (receiver *peerRPCReceiver) SetCredentials(args *SetCredentialsArgs, reply *VoidReply) error {
	if !args.Credentials.IsValid() {
		return fmt.Errorf("invalid credentials passed")
	}

	// Acquire lock before updating global configuration.
	globalServerConfigMu.Lock()
	defer globalServerConfigMu.Unlock()

	// Update credentials in memory
	prevCred := globalServerConfig.SetCredential(args.Credentials)

	// Save credentials to config file
	if err := globalServerConfig.Save(getConfigFile()); err != nil {
		// As saving configurstion failed, restore previous credential in memory.
		globalServerConfig.SetCredential(prevCred)

		logger.LogIf(context.Background(), err)
		return err
	}

	return nil
}

// NewPeerRPCServer - returns new peer RPC server.
func NewPeerRPCServer() (*xrpc.Server, error) {
	rpcServer := xrpc.NewServer()
	if err := rpcServer.RegisterName(peerServiceName, &peerRPCReceiver{}); err != nil {
		return nil, err
	}
	return rpcServer, nil
}

// registerPeerRPCRouter - creates and registers Peer RPC server and its router.
func registerPeerRPCRouter(router *mux.Router) {
	rpcServer, err := NewPeerRPCServer()
	logger.FatalIf(err, "Unable to initialize peer RPC Server", context.Background())
	subrouter := router.PathPrefix(minioReservedBucketPath).Subrouter()
	subrouter.Path(peerServiceSubPath).Handler(rpcServer)
}
