package retriever

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"os/user"
	"regexp"
	"strconv"
	"sync"
	"syscall"
	"text/template"

	"github.com/container-storage-interface/spec/lib/go/csi"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/dell/gocsi"
	csictx "github.com/dell/gocsi/context"
	"github.com/dell/gocsi/utils"
)

// RetrieverServer is the server API for Retriever service.
type RetrieverServer interface {
	GetPVCLabels(context.Context, *GetPVCLabelsRequest) (*GetPVCLabelsResponse, error)
}

type GetPVCLabelsRequest struct {
	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
}
type GetPVCLabelsResponse struct {
	Parameters map[string]string `protobuf:"bytes,4,rep,name=parameters,proto3" json:"parameters,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

// Run launches a CSI storage plug-in.
func Run(
	ctx context.Context,
	appName, appDescription, appUsage string,
	sp RetrieverPluginProvider) {

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
	if os.Getenv(EnvVarEndpoint) == "" {
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

// RetrieverPluginProvider is able to serve a gRPC endpoint that provides
// the CSI services: Retriever
type RetrieverPluginProvider interface {

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

// RetrieverPlugin is the collection of services and data used to server
// a new gRPC endpoint that acts as a CSI storage plug-in (SP).
type RetrieverPlugin struct {
	// MetadataRetriever is the eponymous CSI service.
	MetadataRetriever RetrieverServer

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
	BeforeServe func(context.Context, *RetrieverPlugin, net.Listener) error

	// EnvVars is a list of default environment variables and values.
	EnvVars []string

	// RegisterAdditionalServers allows the driver to register additional
	// grpc servers on the same grpc connection. These can be used
	// for proprietary extensions.
	RegisterAdditionalServers func(*grpc.Server)

	serveOnce sync.Once
	stopOnce  sync.Once
	server    *grpc.Server

	envVars    map[string]string
	pluginInfo csi.GetPluginInfoResponse
}

// Serve accepts incoming connections on the listener lis, creating
// a new ServerTransport and service goroutine for each. The service
// goroutine read gRPC requests and then call the registered handlers
// to reply to them. Serve returns when lis.Accept fails with fatal
// errors.  lis will be closed when this method returns.
// Serve always returns non-nil error.
func (sp *RetrieverPlugin) Serve(ctx context.Context, lis net.Listener) error {
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

		// Initialize the storage plug-in's info.
		//sp.initPluginInfo(ctx)

		// Initialize the interceptors.
		sp.initInterceptors(ctx)

		// Invoke the SP's BeforeServe function to give the SP a chance
		// to perform any local initialization routines.
		if f := sp.BeforeServe; f != nil {
			if err = f(ctx, sp, lis); err != nil {
				return
			}
		}

		// Add the interceptors to the server if any are configured.
		if i := sp.Interceptors; len(i) > 0 {
			sp.ServerOpts = append(sp.ServerOpts,
				grpc.UnaryInterceptor(utils.ChainUnaryServer(i...)))
		}

		// Initialize the gRPC server.
		sp.server = grpc.NewServer(sp.ServerOpts...)

		if sp.MetadataRetriever == nil {
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
func (sp *RetrieverPlugin) Stop(ctx context.Context) {
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
func (sp *RetrieverPlugin) GracefulStop(ctx context.Context) {
	sp.stopOnce.Do(func() {
		if sp.server != nil {
			sp.server.GracefulStop()
		}
		log.Info("gracefully stopped")
	})
}

const netUnix = "unix"

func (sp *RetrieverPlugin) initEndpointPerms(
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

func (sp *RetrieverPlugin) initEndpointOwner(
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

func (sp *RetrieverPlugin) lookupEnv(key string) (string, bool) {
	val, ok := sp.envVars[key]
	return val, ok
}

func (sp *RetrieverPlugin) setenv(key, val string) error {
	sp.envVars[key] = val
	return nil
}

func (sp *RetrieverPlugin) getEnvBool(ctx context.Context, key string) bool {
	v, ok := csictx.LookupEnv(ctx, key)
	if !ok {
		return false
	}
	if b, err := strconv.ParseBool(v); err == nil {
		return b
	}
	return false
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

type logger struct {
	f func(msg string, args ...interface{})
	w io.Writer
}

func newLogger(f func(msg string, args ...interface{})) *logger {
	l := &logger{f: f}
	r, w := io.Pipe()
	l.w = w
	go func() {
		scan := bufio.NewScanner(r)
		for scan.Scan() {
			f(scan.Text())
		}
	}()
	return l
}

func (l *logger) Write(data []byte) (int, error) {
	return l.w.Write(data)
}