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

package retriever

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/user"
	"regexp"
	"strconv"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/dell/csi-metadata-retriever/service"
	"github.com/dell/gocsi"
	csictx "github.com/dell/gocsi/context"
)

// PluginProvider is able to serve a gRPC endpoint that provides
// the CSI services: Retriever
type PluginProvider interface {

	// Serve accepts incoming connections on the listener lis, creating
	// a new ServerTransport and service goroutine for each. The service
	// goroutine read gRPC requests and then call the registered handlers
	// to reply to them. Serve returns when lis.Accept fails with fatal
	// errors.  lis will be closed when this method returns.
	// Serve always returns non-nil error.
	Serve(ctx context.Context, lis net.Listener) error

	// Stop stops the gRPC server. It immediately closes all open
	// connections and listeners.
	// It cancels all active RPCs on the server side and the corresponding
	// pending RPCs on the client side will get notified by connection
	// errors.
	Stop(ctx context.Context)

	// GracefulStop stops the gRPC server gracefully. It stops the server
	// from accepting new connections and RPCs and blocks until all the
	// pending RPCs are finished.
	GracefulStop(ctx context.Context)
}

// Plugin is the collection of services and data used to server
// a new gRPC endpoint that acts as a CSI storage plug-in (SP).
type Plugin struct {
	// MetadataRetriever is the eponymous CSI service.
	MetadataRetrieverService service.Service

	// ServerOpts is a list of gRPC server options used when serving
	// the SP. This list should not include a gRPC interceptor option
	// as one is created automatically based on the interceptor configuration
	// or provided list of interceptors.
	ServerOpts []grpc.ServerOption

	// Interceptors is a list of gRPC server interceptors to use when
	// serving the SP. This list should not include the interceptors
	// defined in the GoCSI package as those are configured by default
	// based on runtime configuration settings.
	Interceptors []grpc.UnaryServerInterceptor

	// BeforeServe is an optional callback that is invoked after the
	// StoragePlugin has been initialized, just prior to the creation
	// of the gRPC server. This callback may be used to perform custom
	// initialization logic, modify the interceptors and server options,
	// or prevent the server from starting by returning a non-nil error.
	BeforeServe func(context.Context, *Plugin, net.Listener) error

	// EnvVars is a list of default environment variables and values.
	EnvVars []string

	// RegisterAdditionalServers allows the driver to register additional
	// grpc servers on the same grpc connection. These can be used
	// for proprietary extensions.
	RegisterAdditionalServers func(*grpc.Server)

	serveOnce sync.Once
	stopOnce  sync.Once
	server    *grpc.Server

	envVars map[string]string
}

// Serve accepts incoming connections on the listener lis, creating
// a new ServerTransport and service goroutine for each. The service
// goroutine read gRPC requests and then call the registered handlers
// to reply to them. Serve returns when lis.Accept fails with fatal
// errors.  lis will be closed when this method returns.
// Serve always returns non-nil error.
func (sp *Plugin) Serve(ctx context.Context, lis net.Listener) error {
	var err error
	sp.serveOnce.Do(func() {
		// Please note that the order of the below init functions is
		// important and should not be altered unless by someone aware
		// of how they work.

		// Adding this function to the context allows `csictx.LookupEnv`
		// to search this SP's default env vars for a value.
		ctx = csictx.WithLookupEnv(ctx, sp.lookupEnv)

		// Adding this function to the context allows `csictx.Setenv`
		// to set environment variables in this SP's env var store.
		ctx = csictx.WithSetenv(ctx, sp.setenv)

		// Initialize the storage plug-in's environment variables map.
		sp.initEnvVars(ctx)

		// Adjust the endpoint's file permissions.
		if err = sp.initEndpointPerms(ctx, lis); err != nil {
			return
		}

		// Adjust the endpoint's file ownership.
		if err = sp.initEndpointOwner(ctx, lis); err != nil {
			return
		}

		// Invoke the SP's BeforeServe function to give the SP a chance
		// to perform any local initialization routines.
		if f := sp.BeforeServe; f != nil {
			if err = f(ctx, sp, lis); err != nil {
				return
			}
		}

		// Initialize the gRPC server.
		sp.server = grpc.NewServer(sp.ServerOpts...)

		if sp.MetadataRetrieverService == nil {
			err = errors.New("retriever service is required")
			return
		}

		// Register any additional servers required.
		if sp.RegisterAdditionalServers != nil {
			sp.RegisterAdditionalServers(sp.server)
		}

		endpoint := fmt.Sprintf(
			"%s://%s",
			lis.Addr().Network(), lis.Addr().String())
		log.WithField("endpoint", endpoint).Info("serving")

		// Start the gRPC server.
		err = sp.server.Serve(lis)
		return
	})
	return err
}

// Stop stops the gRPC server. It immediately closes all open
// connections and listeners.
// It cancels all active RPCs on the server side and the corresponding
// pending RPCs on the client side will get notified by connection
// errors.
func (sp *Plugin) Stop(ctx context.Context) {
	sp.stopOnce.Do(func() {
		if sp.server != nil {
			sp.server.Stop()
		}
		log.Info("stopped")
	})
}

// GracefulStop stops the gRPC server gracefully. It stops the server
// from accepting new connections and RPCs and blocks until all the
// pending RPCs are finished.
func (sp *Plugin) GracefulStop(ctx context.Context) {
	sp.stopOnce.Do(func() {
		if sp.server != nil {
			sp.server.GracefulStop()
		}
		log.Info("gracefully stopped")
	})
}

const netUnix = "unix"

func (sp *Plugin) initEndpointPerms(
	ctx context.Context, lis net.Listener) error {

	if lis.Addr().Network() != netUnix {
		return nil
	}

	v, ok := csictx.LookupEnv(ctx, gocsi.EnvVarEndpointPerms)
	if !ok || v == "0755" {
		return nil
	}
	u, err := strconv.ParseUint(v, 8, 32)
	if err != nil {
		return err
	}

	p := lis.Addr().String()
	m := os.FileMode(u)

	log.WithFields(map[string]interface{}{
		"path": p,
		"mode": m,
	}).Info("chmod csi endpoint")

	if err := os.Chmod(p, m); err != nil {
		return err
	}

	return nil
}

func (sp *Plugin) initEndpointOwner(
	ctx context.Context, lis net.Listener) error {

	if lis.Addr().Network() != netUnix {
		return nil
	}

	var (
		usrName string
		grpName string

		uid  = os.Getuid()
		gid  = os.Getgid()
		puid = uid
		pgid = gid
	)

	if v, ok := csictx.LookupEnv(ctx, gocsi.EnvVarEndpointUser); ok {
		m, err := regexp.MatchString(`^\d+$`, v)
		if err != nil {
			return err
		}
		usrName = v
		szUID := v
		if m {
			u, err := user.LookupId(v)
			if err != nil {
				return err
			}
			usrName = u.Username
		} else {
			u, err := user.Lookup(v)
			if err != nil {
				return err
			}
			szUID = u.Uid
		}
		iuid, err := strconv.Atoi(szUID)
		if err != nil {
			return err
		}
		uid = iuid
	}

	if v, ok := csictx.LookupEnv(ctx, gocsi.EnvVarEndpointGroup); ok {
		m, err := regexp.MatchString(`^\d+$`, v)
		if err != nil {
			return err
		}
		grpName = v
		szGID := v
		if m {
			u, err := user.LookupGroupId(v)
			if err != nil {
				return err
			}
			grpName = u.Name
		} else {
			u, err := user.LookupGroup(v)
			if err != nil {
				return err
			}
			szGID = u.Gid
		}
		igid, err := strconv.Atoi(szGID)
		if err != nil {
			return err
		}
		gid = igid
	}

	if uid != puid || gid != pgid {
		f := lis.Addr().String()
		log.WithFields(map[string]interface{}{
			"uid":  usrName,
			"gid":  grpName,
			"path": f,
		}).Info("chown csi endpoint")
		if err := os.Chown(f, uid, gid); err != nil {
			return err
		}
	}

	return nil
}

func (sp *Plugin) lookupEnv(key string) (string, bool) {
	val, ok := sp.envVars[key]
	return val, ok
}

func (sp *Plugin) setenv(key, val string) error {
	sp.envVars[key] = val
	return nil
}

func (sp *Plugin) initEnvVars(ctx context.Context) {

	// Copy the environment variables from the public EnvVar
	// string slice to the private envVars map for quick lookup.
	sp.envVars = map[string]string{}
	for _, v := range sp.EnvVars {
		// Environment variables must adhere to one of the following
		// formats:
		//
		//     - ENV_VAR_KEY=
		//     - ENV_VAR_KEY=ENV_VAR_VAL
		pair := strings.SplitN(v, "=", 2)
		if len(pair) < 1 || len(pair) > 2 {
			continue
		}

		// Ensure the environment variable is stored in all upper-case
		// to make subsequent map-lookups deterministic.
		key := strings.ToUpper(pair[0])

		// Check to see if the value for the key is available from the
		// context's os.Environ or os.LookupEnv functions. If neither
		// return a value then use the provided default value.
		var val string
		if v, ok := csictx.LookupEnv(ctx, key); ok {
			val = v
		} else if len(pair) > 1 {
			val = pair[1]
		}
		sp.envVars[key] = val
	}

	// Check for the debug value.
	if v, ok := csictx.LookupEnv(ctx, gocsi.EnvVarDebug); ok {
		/* #nosec G104 */
		if ok, _ := strconv.ParseBool(v); ok {
			csictx.Setenv(ctx, gocsi.EnvVarReqLogging, "true")
			csictx.Setenv(ctx, gocsi.EnvVarRepLogging, "true")
		}
	}

	return
}
