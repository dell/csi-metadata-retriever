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

package main

import (
	"context"
	"flag"
	"net"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"text/template"

	log "github.com/sirupsen/logrus"

	"github.com/dell/csi-metadata-retriever/provider"
	"github.com/dell/csi-metadata-retriever/retriever"
	"github.com/dell/csi-metadata-retriever/utils"
	"github.com/dell/gocsi"
	csictx "github.com/dell/gocsi/context"
)

// main is ignored when this package is built as a go plug-in.
func main() {
	Run(
		context.Background(),
		"MetadataRetriever",
		"A description of the SP",
		"",
		provider.New())
}

const netUnix = "unix"

var printUsage = func(appName, appDescription, appUsage, binPath string) {
	// app is the information passed to the printUsage function
	app := struct {
		Name        string
		Description string
		Usage       string
		BinPath     string
	}{
		appName,
		appDescription,
		appUsage,
		binPath,
	}

	t, err := template.New("t").Parse(usage)
	if err != nil {
		log.WithError(err).Fatalln("failed to parse usage template")
	}
	if err := t.Execute(os.Stderr, app); err != nil {
		log.WithError(err).Fatalln("failed emitting usage")
	}
	return
}

var rmSockFileOnce sync.Once
var rmSockFile = func(l net.Listener) {
	rmSockFileOnce.Do(func() {
		if l == nil || l.Addr() == nil {
			return
		}
		/* #nosec G104 */
		if l.Addr().Network() == netUnix {
			sockFile := l.Addr().String()
			err := os.RemoveAll(sockFile)
			if err != nil {
				log.Warnf("failed to remove sock file: %s", err)
			}
			log.WithField("path", sockFile).Info("removed sock file")
		}
	})
}

// Run launches a CSI storage plug-in.
func Run(
	ctx context.Context,
	appName, appDescription, appUsage string,
	sp retriever.PluginProvider,
) {
	// Check for the debug value.
	if v, ok := csictx.LookupEnv(ctx, gocsi.EnvVarDebug); ok {
		/* #nosec G104 */
		if ok, _ := strconv.ParseBool(v); ok {
			err := csictx.Setenv(ctx, gocsi.EnvVarLogLevel, "debug")
			if err != nil {
				log.Warnf("failed to set EnvVarLogLevel")
			}
			err = csictx.Setenv(ctx, gocsi.EnvVarReqLogging, "true")
			if err != nil {
				log.Warnf("failed to set EnvVarReqLogging")
			}
			err = csictx.Setenv(ctx, gocsi.EnvVarRepLogging, "true")
			if err != nil {
				log.Warnf("failed to set EnvVarRepLogging")
			}
		}
	}

	// Adjust the log level.
	lvl := log.InfoLevel
	if v, ok := csictx.LookupEnv(ctx, gocsi.EnvVarLogLevel); ok {
		var err error
		if lvl, err = log.ParseLevel(v); err != nil {
			lvl = log.InfoLevel
		}
	}
	log.SetLevel(lvl)

	// Check for a help flag.
	fs := flag.NewFlagSet("csp", flag.ExitOnError)
	// fs.Usage = printUsage(appName, appDescription, appUsage, os.Args[0])
	var help bool
	fs.BoolVar(&help, "?", false, "")
	err := fs.Parse(os.Args)
	if err == flag.ErrHelp || help {
		printUsage(appName, appDescription, appUsage, os.Args[0])
		os.Exit(1)
	}

	// If no endpoint is set then print the usage.
	if os.Getenv(utils.EnvVarEndpoint) == "" {
		printUsage(appName, appDescription, appUsage, os.Args[0])
		os.Exit(1)
	}

	l, err := utils.GetCSIEndpointListener()
	if err != nil {
		log.WithError(err).Fatalln("failed to listen")
	}

	// Define a lambda that can be used in the exit handler
	// to remove a potential UNIX sock file.

	trapSignals(func() {
		sp.GracefulStop(ctx)
		rmSockFile(l)
		log.Info("server stopped gracefully")
	}, func() {
		sp.Stop(ctx)
		rmSockFile(l)
		log.Info("server aborted")
	})

	if err := sp.Serve(ctx, l); err != nil {
		rmSockFile(l)
		log.WithError(err).Fatal("grpc failed")
	}
}

func trapSignals(onExit, onAbort func()) {
	sigc := make(chan os.Signal, 1)
	sigs := []os.Signal{
		syscall.SIGTERM,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGQUIT,
	}
	signal.Notify(sigc, sigs...)
	go func() {
		for s := range sigc {
			ok, graceful := isExitSignal(s)
			if !ok {
				continue
			}
			if !graceful {
				log.WithField("signal", s).Error("received signal; aborting")
				if onAbort != nil {
					onAbort()
				}
				os.Exit(1)
			}
			log.WithField("signal", s).Info("received signal; shutting down")
			if onExit != nil {
				onExit()
			}
			os.Exit(0)
		}
	}()
}

// isExitSignal returns a flag indicating whether a signal SIGHUP,
// SIGINT, SIGTERM, or SIGQUIT. The second return value is whether it is a
// graceful exit. This flag is true for SIGTERM, SIGHUP, SIGINT, and SIGQUIT.
func isExitSignal(s os.Signal) (bool, bool) {
	switch s {
	case syscall.SIGTERM,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGQUIT:
		return true, true
	default:
		return false, false
	}
}
