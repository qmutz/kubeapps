/*
Copyright 2021 VMware. All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package server

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/soheilhy/cmux"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	plugins "github.com/kubeapps/kubeapps/cmd/kubeapps-apis/gen/core/plugins/v1alpha1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	log "k8s.io/klog/v2"
)

// Serve is the root command that is run when no other sub-commands are present.
// It runs the gRPC service, registering the configured plugins.
func Serve(port int, pluginDirs []string) {
	// Create the grpc server and register the reflection server (for now, useful for discovery
	// using grpcurl) or similar.
	grpcSrv := grpc.NewServer()
	reflection.Register(grpcSrv)

	// Create the http server, register our core service followed by any plugins.
	listenAddr := fmt.Sprintf(":%d", port)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	gwArgs := gwHandlerArgs{
		ctx:         ctx,
		mux:         runtime.NewServeMux(),
		addr:        listenAddr,
		dialOptions: []grpc.DialOption{grpc.WithInsecure()},
	}
	httpSrv := &http.Server{
		Handler: gwArgs.mux,
	}

	// Create the core.plugins server which handles registration of plugins,
	// and register it for both grpc and http.
	pluginsServer, err := NewPluginsServer(pluginDirs, grpcSrv, gwArgs)
	if err != nil {
		log.Fatalf("failed to initialize plugins server: %v", err)
	}
	plugins.RegisterPluginsServiceServer(grpcSrv, pluginsServer)
	err = plugins.RegisterPluginsServiceHandlerFromEndpoint(gwArgs.ctx, gwArgs.mux, gwArgs.addr, gwArgs.dialOptions)
	if err != nil {
		log.Fatalf("failed to register core.plugins handler for gateway: %v", err)
	}

	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// Multiplex the connection between grpc and http.
	// Note: due to a change in the grpc protocol, it's no longer possible to just match
	// on the simpler cmux.HTTP2HeaderField("content-type", "application/grpc"). More details
	// at https://github.com/soheilhy/cmux/issues/64
	mux := cmux.New(lis)
	grpcLis := mux.MatchWithWriters(cmux.HTTP2MatchHeaderFieldSendSettings("content-type", "application/grpc"))
	httpLis := mux.Match(cmux.Any())

	go grpcSrv.Serve(grpcLis)
	go httpSrv.Serve(httpLis)

	log.Infof("Starting server on :%d", port)
	if err := mux.Serve(); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

// gwHandlerArgs is a helper struct just encapsulating all the args
// required when registering an HTTP handler for the gateway.
type gwHandlerArgs struct {
	ctx         context.Context
	mux         *runtime.ServeMux
	addr        string
	dialOptions []grpc.DialOption
}
