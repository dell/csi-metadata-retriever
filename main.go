package main

import (
	"context"
	"flag"
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

// Run launches a CSI storage plug-in.
func Run(
	ctx context.Context,
	appName, appDescription, appUsage string,
	sp retriever.PluginProvider) {

	// Check for the debug value.
	if v, ok := csictx.LookupEnv(ctx, gocsi.EnvVarDebug); ok {
		/* #nosec G104 */
		if ok, _ := strconv.ParseBool(v); ok {
			csictx.Setenv(ctx, gocsi.EnvVarLogLevel, "debug")
			csictx.Setenv(ctx, gocsi.EnvVarReqLogging, "true")
			csictx.Setenv(ctx, gocsi.EnvVarRepLogging, "true")
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

	printUsage := func() {
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
			os.Args[0],
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

	// Check for a help flag.
	fs := flag.NewFlagSet("csp", flag.ExitOnError)
	fs.Usage = printUsage
	var help bool
	fs.BoolVar(&help, "?", false, "")
	err := fs.Parse(os.Args)
	if err == flag.ErrHelp || help {
		printUsage()
		os.Exit(1)
	}

	// If no endpoint is set then print the usage.
	if os.Getenv(utils.EnvVarEndpoint) == "" {
		printUsage()
		os.Exit(1)
	}

	l, err := utils.GetCSIEndpointListener()
	if err != nil {
		log.WithError(err).Fatalln("failed to listen")
	}

	// Define a lambda that can be used in the exit handler
	// to remove a potential UNIX sock file.
	var rmSockFileOnce sync.Once
	rmSockFile := func() {
		rmSockFileOnce.Do(func() {
			if l == nil || l.Addr() == nil {
				return
			}
			/* #nosec G104 */
			if l.Addr().Network() == netUnix {
				sockFile := l.Addr().String()
				os.RemoveAll(sockFile)
				log.WithField("path", sockFile).Info("removed sock file")
			}
		})
	}

	trapSignals(func() {
		sp.GracefulStop(ctx)
		rmSockFile()
		log.Info("server stopped gracefully")
	}, func() {
		sp.Stop(ctx)
		rmSockFile()
		log.Info("server aborted")
	})

	if err := sp.Serve(ctx, l); err != nil {
		rmSockFile()
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
