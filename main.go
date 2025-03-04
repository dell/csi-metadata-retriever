/*
 *
 * Copyright © 2022-2025 Dell Inc. or its subsidiaries. All Rights Reserved.
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
	"io"
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

const netUnix = "unix"

var (
	exit          = os.Exit
	parseTemplate = func(usage string) (*template.Template, error) {
		return template.New("t").Parse(usage)
	}
)

var executeTemplate = func(t *template.Template, wr io.Writer, data interface{}) error {
	return t.Execute(wr, data)
}

func main() {
	runMain(provider.New())
}

func runMain(sp retriever.PluginProvider) {
	var ctx context.Context
	ctx = context.Background()
	appName := "MetadataRetriever"
	appDescription := "A description of the SP"
	appUsage := ""
	Run(ctx, appName, appDescription, appUsage, sp)
}

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

	t, err := parseTemplate(usage)
	if err != nil {
		log.WithError(err).Fatalln("failed to parse usage template")
	}
	if err := executeTemplate(t, os.Stderr, app); err != nil {
		log.WithError(err).Fatalln("failed emitting usage")
	}
	return
}

var (
	rmSockFileOnce sync.Once
	rmSockFile     = func(l net.Listener) {
		rmSockFileOnce.Do(func() {
			if l == nil {
				log.Info("listener is nil")
				return
			}
			addr := l.Addr()
			if addr == nil {
				log.Info("listener address is nil")
				return
			}
			log.Infof("listener address: %v", l.Addr().String())
			/* #nosec G104 */
			if l.Addr().Network() == netUnix {
				sockAddress := l.Addr()
				sockFile := sockAddress.String()
				log.Infof("removing socket file: %s", sockFile)
				err := os.RemoveAll(sockFile)
				if err != nil {
					log.Warnf("failed to remove sock file: %s", err)
				}
				log.WithField("path", sockFile).Info("removed sock file")
			}
		})
	}
)

var (
	getCSIEndpointListener = utils.GetCSIEndpointListener
	setenv                 = csictx.Setenv
	lookupEnv              = csictx.LookupEnv
)

// Run launches a CSI storage plug-in.
func Run(
	ctx context.Context,
	appName, appDescription, appUsage string,
	sp retriever.PluginProvider,
) {
	// Check for the debug value.
	if v, ok := lookupEnv(ctx, gocsi.EnvVarDebug); ok {
		/* #nosec G104 */
		if ok, _ := strconv.ParseBool(v); ok {
			log.Infof("setting EnvVarLogLevel")
			err := setenv(ctx, gocsi.EnvVarLogLevel, "debug")
			if err != nil {
				log.Warnf("failed to set EnvVarLogLevel")
			}
			log.Infof("setting EnvVarReqLogging")
			err = setenv(ctx, gocsi.EnvVarReqLogging, "true")
			if err != nil {
				log.Warnf("failed to set EnvVarReqLogging")
			}
			log.Infof("setting EnvVarRepLogging")
			err = setenv(ctx, gocsi.EnvVarRepLogging, "true")
			if err != nil {
				log.Warnf("failed to set EnvVarRepLogging")
			}
		}
	}

	// Adjust the log level.
	lvl := log.InfoLevel
	if v, ok := lookupEnv(ctx, gocsi.EnvVarLogLevel); ok {
		var err error
		if lvl, err = log.ParseLevel(v); err != nil {
			lvl = log.InfoLevel
		}
	}
	log.Info("setting log level to: ", lvl)
	log.SetLevel(lvl)

	// Check for a help flag.
	fs := flag.NewFlagSet("csp", flag.ExitOnError)
	// fs.Usage = printUsage(appName, appDescription, appUsage, os.Args[0])
	var help bool
	fs.BoolVar(&help, "?", false, "")
	err := fs.Parse(os.Args)
	if err == flag.ErrHelp || help {
		printUsage(appName, appDescription, appUsage, os.Args[0])
		exit(1)
	}

	// If no endpoint is set then print the usage.
	if os.Getenv(utils.EnvVarEndpoint) == "" {
		log.Warnf("no endpoint set")
		printUsage(appName, appDescription, appUsage, os.Args[0])
		exit(1)
	}

	l, err := getCSIEndpointListener()
	if err != nil {
		log.WithError(err).Fatalln("failed to listen")
	}

	trapSignals(func() {
		sp.GracefulStop(ctx)
		rmSockFile(l)
		log.Info("server stopped gracefully")
	})

	if err := sp.Serve(ctx, l); err != nil {
		rmSockFile(l)
		log.WithError(err).Fatal("grpc failed")
	}
}

func trapSignals(onExit func()) {
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
			log.Printf("received signal: %v", s)
			ok, graceful := isExitSignal(s)
			log.Printf("isExitSignal: ok=%v, graceful=%v", ok, graceful)
			if !ok {
				continue
			}
			log.Printf("received signal; shutting down: %v", s)

			if onExit != nil {
				onExit()
			}
			exit(0)
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
