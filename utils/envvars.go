package utils

import (
	"context"
	"strconv"
	"strings"

	"github.com/dell/gocsi"
	csictx "github.com/dell/gocsi/context"
)

const (
	// EnvVarEndpoint is the name of the environment variable used to
	// specify the CSI endpoint.
	EnvVarEndpoint = "CSI_RETRIEVER_ENDPOINT"

	// EnvVarEndpointPerms is the name of the environment variable used
	// to specify the file permissions for the CSI endpoint when it is
	// a UNIX socket file. This setting has no effect if CSI_ENDPOINT
	// specifies a TCP socket. The default value is 0755.
	//EnvVarEndpointPerms = "X_CSI_ENDPOINT_PERMS"				use from gocsi

	// EnvVarEndpointUser is the name of the environment variable used
	// to specify the UID or name of the user that owns the endpoint's
	// UNIX socket file. This setting has no effect if CSI_ENDPOINT
	// specifies a TCP socket. The default value is the user that starts
	// the process.
	//EnvVarEndpointUser = "X_CSI_ENDPOINT_USER"

	// EnvVarEndpointGroup is the name of the environment variable used
	// to specify the GID or name of the group that owns the endpoint's
	// UNIX socket file. This setting has no effect if CSI_ENDPOINT
	// specifies a TCP socket. The default value is the group that starts
	// the process.
	//EnvVarEndpointGroup = "X_CSI_ENDPOINT_GROUP"

	// EnvVarDebug is the name of the environment variable used to
	// determine whether or not debug mode is enabled.
	//
	// Setting this environment variable to a truthy value is the
	// equivalent of X_CSI_LOG_LEVEL=DEBUG, X_CSI_REQ_LOGGING=true,
	// and X_CSI_REP_LOGGING=true.
	//EnvVarDebug = "X_CSI_DEBUG"

	// EnvVarLogLevel is the name of the environment variable used to
	// specify the log level. Valid values include PANIC, FATAL, ERROR,
	// WARN, INFO, and DEBUG.
	//EnvVarLogLevel = "X_CSI_LOG_LEVEL"

	// EnvVarPluginInfo is the name of the environment variable used to
	// specify the plug-in info in the format:
	//
	//         NAME, VENDOR_VERSION[, MANIFEST...]
	//
	// The MANIFEST value may be a series of additional comma-separated
	// key/value pairs.
	//
	// Please see the encoding/csv package (https://goo.gl/1j1xb9) for
	// information on how to quote keys and/or values to include leading
	// and trailing whitespace.
	//
	// Setting this environment variable will cause the program to
	// bypass the SP's GetPluginInfo RPC and returns the specified
	// information instead.
	//EnvVarPluginInfo = "X_CSI_PLUGIN_INFO"

	// EnvVarReqLogging is the name of the environment variable
	// used to determine whether or not to enable request logging.
	//
	// Setting this environment variable to a truthy value enables
	// request logging to STDOUT.
	//EnvVarReqLogging = "X_CSI_REQ_LOGGING"

	// EnvVarRepLogging is the name of the environment variable
	// used to determine whether or not to enable response logging.
	//
	// Setting this environment variable to a truthy value enables
	// response logging to STDOUT.
	//EnvVarRepLogging = "X_CSI_REP_LOGGING"

	// EnvVarReqIDInjection is the name of the environment variable
	// used to determine whether or not to enable request ID injection.
	//EnvVarReqIDInjection = "X_CSI_REQ_ID_INJECTION"

	// EnvVarSpecValidation is the name of the environment variable
	// used to determine whether or not to enable validation of CSI
	// request and response messages. Setting X_CSI_SPEC_VALIDATION=true
	// is the equivalent to setting X_CSI_SPEC_REQ_VALIDATION=true and
	// X_CSI_SPEC_REP_VALIDATION=true.
	//EnvVarSpecValidation = "X_CSI_SPEC_VALIDATION"

	// EnvVarSpecReqValidation is the name of the environment variable
	// used to determine whether or not to enable validation of CSI request
	// messages.
	//EnvVarSpecReqValidation = "X_CSI_SPEC_REQ_VALIDATION"

	// EnvVarSpecRepValidation is the name of the environment variable
	// used to determine whether or not to enable validation of CSI response
	// messages. Invalid responses are marshalled into a gRPC error with
	// a code of "Internal."
	//EnvVarSpecRepValidation = "X_CSI_SPEC_REP_VALIDATION"

	// EnvVarDisableFieldLen is the name of the environment variable used
	// to determine whether or not to disable validation of CSI request and
	// response field lengths against the permitted lenghts defined in the spec
	//EnvVarDisableFieldLen = "X_CSI_SPEC_DISABLE_LEN_CHECK"

	// EnvVarCreds is the name of the environment variable
	// used to determine whether or not user credentials are required for
	// all RPCs. This value may be overridden for specific RPCs.
	/* #nosec G101 */
	//EnvVarCreds = "X_CSI_REQUIRE_CREDS"

)

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
