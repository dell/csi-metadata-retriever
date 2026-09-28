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
	"sync"
	"syscall"
	"text/template"

	"github.com/dell/csi-metadata-retriever/csiendpoint"
	"github.com/dell/csi-metadata-retriever/provider"
	"github.com/dell/csi-metadata-retriever/retriever"
	"github.com/dell/csmlog"
)

const netUnix = "unix"

var (
	getCSIEndpointListener = csiendpoint.GetCSIEndpointListener
	exit                   = os.Exit
	parseTemplate          = func(usage string) (*template.Template, error) {
		return template.New("t").Parse(usage)
	}
	executeTemplate = func(t *template.Template, wr io.Writer, data interface{}) error {
		return t.Execute(wr, data)
	}
	rmSockFileOnce sync.Once
)

var rmSockFile = func(l net.Listener) {
	rmSockFileOnce.Do(func() {
		if l == nil {
			csmlog.Info("rmSockFile: listener is nil")
			return
		}
		addr := l.Addr()
		if addr == nil {
			csmlog.Info("rmSockFile: listener address is nil")
			return
		}
		csmlog.Infof("listener address: %v", l.Addr().String())
		/* #nosec G104 */
		if l.Addr().Network() == netUnix {
			sockAddress := l.Addr()
			sockFile := sockAddress.String()
			csmlog.WithFields(csmlog.Fields{
				csmlog.FieldComponent: "retriever",
				csmlog.FieldOperation: "rmSockFile",
				csmlog.FieldProtocol:  l.Addr().Network(),
				"path":                sockFile,
			}).Info("removing socket file")
			err := os.RemoveAll(sockFile)
			if err != nil {
				csmlog.WithFields(csmlog.Fields{
					csmlog.FieldComponent: "retriever",
					csmlog.FieldOperation: "rmSockFile",
					csmlog.FieldProtocol:  l.Addr().Network(),
					csmlog.FieldError:     err.Error(),
					"path":                sockFile,
				}).Warn("failed to remove sock file")
			}
			csmlog.WithFields(csmlog.Fields{
				csmlog.FieldComponent: "retriever",
				csmlog.FieldOperation: "rmSockFile",
				csmlog.FieldProtocol:  l.Addr().Network(),
				"path":                sockFile,
			}).Info("removed sock file")
		}
	})
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
		csmlog.WithFields(csmlog.Fields{
			csmlog.FieldComponent: "retriever",
			csmlog.FieldOperation: "printUsage",
			csmlog.FieldError:     err.Error(),
		}).Fatal("failed to parse usage template")
	}
	err = executeTemplate(t, os.Stderr, app)
	if err != nil {
		csmlog.WithFields(csmlog.Fields{
			csmlog.FieldComponent: "retriever",
			csmlog.FieldOperation: "printUsage",
			csmlog.FieldError:     err.Error(),
		}).Fatal("failed emitting usage")
	}
}

func main() {
	runMain(provider.New())
}

func runMain(sp retriever.PluginProvider) {
	ctx := context.Background()
	appName := "MetadataRetriever"
	appDescription := "A description of the SP"
	appUsage := ""
	Run(ctx, appName, appDescription, appUsage, sp)
}

// Run launches a CSI storage plug-in.
func Run(
	ctx context.Context,
	appName, appDescription, appUsage string,
	sp retriever.PluginProvider,
) {
	// Check for a help flag.
	fs := flag.NewFlagSet("csp", flag.ExitOnError)
	fs.Usage = func() { printUsage(appName, appDescription, appUsage, os.Args[0]) }
	var help bool
	fs.BoolVar(&help, "?", false, "")
	err := fs.Parse(os.Args)
	if err == flag.ErrHelp || help {
		printUsage(appName, appDescription, appUsage, os.Args[0])
		exit(1)
	}

	// If no endpoint is set then print the usage.
	if os.Getenv(csiendpoint.EnvVarEndpoint) == "" {
		csmlog.Warnf("no endpoint set")
		printUsage(appName, appDescription, appUsage, os.Args[0])
		exit(1)
	}

	l, err := getCSIEndpointListener()
	if err != nil {
		csmlog.WithFields(csmlog.Fields{
			csmlog.FieldComponent: "retriever",
			csmlog.FieldOperation: "startup",
			csmlog.FieldError:     err.Error(),
		}).Error("failed to listen")
		exit(1)
	}

	trapSignals(func() {
		sp.GracefulStop(ctx)
		rmSockFile(l)
		csmlog.WithFields(csmlog.Fields{
			csmlog.FieldComponent: "retriever",
			csmlog.FieldOperation: "shutdown",
			csmlog.FieldProtocol:  l.Addr().Network(),
			"address":             l.Addr().String(),
		}).Info("server stopped gracefully")
	})

	err = sp.Serve(ctx, l)
	if err != nil {
		rmSockFile(l)
		csmlog.WithFields(csmlog.Fields{
			csmlog.FieldComponent: "retriever",
			csmlog.FieldOperation: "startup",
			csmlog.FieldProtocol:  "grpc",
			csmlog.FieldError:     err.Error(),
		}).Error("grpc failed")
		exit(1)
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
	exitFunc := exit
	go func() {
		for s := range sigc {
			csmlog.Infof("received signal: %s", s.String())
			ok, graceful := isExitSignal(s)
			csmlog.Debugf("isExitSignal: is_exit=%v, graceful=%v", ok, graceful)
			if !ok {
				continue
			}
			csmlog.WithFields(csmlog.Fields{
				csmlog.FieldComponent: "retriever",
				csmlog.FieldOperation: "shutdown",
				"signal":              s.String(),
			}).Info("received exit signal; shutting down")

			if onExit != nil {
				onExit()
			}
			exitFunc(0)
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
