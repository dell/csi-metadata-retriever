/*
 *
 * Copyright © 2022 Dell Inc. or its subsidiaries. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *      http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package retriever_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/dell/csi-metadata-retriever/retriever"
	"github.com/dell/csi-metadata-retriever/service"
	"github.com/dell/csi-metadata-retriever/utils"
	"google.golang.org/grpc"
)

var grpcClient *grpc.ClientConn

func TestServer_StartGracefulStop(_ *testing.T) {
	var stop func()
	os.Setenv("CSI_RETRIEVER_ENDPOINT", "/tmp/csi_retriever_test.sock")

	ctx := context.Background()
	sp := new(retriever.Plugin)
	sp.MetadataRetrieverService = service.New()

	fmt.Printf("calling startServer")
	grpcClient, stop = startServer(ctx, sp, true)
	fmt.Printf("back from startServer")
	time.Sleep(5 * time.Second)

	stop()
}

func TestServer_StartStop(_ *testing.T) {
	var stop func()
	os.Setenv("CSI_RETRIEVER_ENDPOINT", "/tmp/csi_retriever_test.sock")

	ctx := context.Background()
	sp := new(retriever.Plugin)
	sp.MetadataRetrieverService = service.New()

	fmt.Printf("calling startServer")
	grpcClient, stop = startServer(ctx, sp, false)
	fmt.Printf("back from startServer")
	time.Sleep(5 * time.Second)

	stop()
}

func startServer(ctx context.Context, sp *retriever.Plugin, gracefulStop bool) (*grpc.ClientConn, func()) {
	lis, err := utils.GetCSIEndpointListener()
	if err != nil {
		fmt.Printf("couldn't open listener: %s\n", err.Error())
		return nil, nil
	}

	fmt.Printf("lis: %v\n", lis)
	go func() {
		fmt.Printf("starting server\n")
		if err := sp.Serve(ctx, lis); err != nil {
			fmt.Printf("http: Server closed. Error: %v", err)
		}
	}()
	network, addr, err := utils.GetCSIEndpoint()
	if err != nil {
		return nil, nil
	}
	fmt.Printf("network %v addr %v\n", network, addr)

	clientOpts := []grpc.DialOption{
		grpc.WithInsecure(),
	}

	// Create a client for the piped connection.
	fmt.Printf("calling gprc.DialContext, ctx %v, addr %s, clientOpts %v\n", ctx, addr, clientOpts)
	client, err := grpc.DialContext(ctx, "unix:"+addr, clientOpts...)
	if err != nil {
		fmt.Printf("DialContext returned error: %s", err.Error())
	}
	fmt.Printf("grpc.DialContext returned ok\n")

	if gracefulStop {
		return client, func() {
			client.Close()
			sp.GracefulStop(ctx)
		}
	}

	return client, func() {
		client.Close()
		sp.Stop(ctx)
	}
}
