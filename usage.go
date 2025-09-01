/*
 *
 * Copyright © 2021-2024 Dell Inc. or its subsidiaries. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

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

const usage = `NAME
    {{.Name}} -- {{.Description}}

SYNOPSIS
    {{.BinPath}}
{{if .Usage}}
STORAGE OPTIONS
{{.Usage}}{{end}}
GLOBAL OPTIONS
    CSI_RETRIEVER_ENDPOINT
        The CSI endpoint may also be specified by the environment variable
        CSI_RETRIEVER_ENDPOINT. The endpoint should adhere to Go's network address
        pattern:

            * tcp://host:port
            * unix:///path/to/file.sock.

        If the network type is omitted then the value is assumed to be an
        absolute or relative filesystem path to a UNIX socket file


    X_CSI_ENDPOINT_PERMS
        When CSI_ENDPOINT is set to a UNIX socket file this environment
        variable may be used to specify the socket's file permissions
        as an octal number, ex. 0644. Please note this value has no
        effect if CSI_ENDPOINT specifies a TCP socket.

        The default value is 0755.

    X_CSI_ENDPOINT_USER
        When CSI_ENDPOINT is set to a UNIX socket file this environment
        variable may be used to specify the UID or user name of the
        user that owns the file. Please note this value has no
        effect if CSI_ENDPOINT specifies a TCP socket.

        If no value is specified then the user owner of the file is the
        same as the user that starts the process.

    X_CSI_ENDPOINT_GROUP
        When CSI_ENDPOINT is set to a UNIX socket file this environment
        variable may be used to specify the GID or group name of the
        group that owns the file. Please note this value has no
        effect if CSI_ENDPOINT specifies a TCP socket.

        If no value is specified then the group owner of the file is the
        same as the group that starts the process.

    X_CSI_DEBUG
        Enabling this option is the same as:
            X_CSI_LOG_LEVEL=debug
            X_CSI_REQ_LOGGING=true
            X_CSI_REP_LOGGING=true

    X_CSI_LOG_LEVEL
        The log level. Valid values include:
           * PANIC
           * FATAL
           * ERROR
           * WARN
           * INFO
           * DEBUG

        The default value is WARN.

    X_CSI_PLUGIN_INFO
        The plug-in information is specified via the following
        comma-separated format:

            NAME, VENDOR_VERSION[, MANIFEST...]

        The MANIFEST value may be a series of additional
        comma-separated key/value pairs.

        Please see the encoding/csv package (https://goo.gl/1j1xb9) for
        information on how to quote keys and/or values to include
        leading and trailing whitespace.

        Setting this environment variable will cause the program to
        bypass the SP's GetPluginInfo RPC and returns the specified
        information instead.

    X_CSI_REQ_LOGGING
        A flag that enables logging of incoming requests to STDOUT.

        Enabling this option sets X_CSI_REQ_ID_INJECTION=true.

    X_CSI_REP_LOGGING
        A flag that enables logging of outgoing responses to STDOUT.

        Enabling this option sets X_CSI_REQ_ID_INJECTION=true.

    X_CSI_REQ_ID_INJECTION
        A flag that enables request ID injection. The ID is parsed from
        the incoming request's metadata with a key of "csi.requestid".
        If no value for that key is found then a new request ID is
        generated using an atomic sequence counter.

    X_CSI_SPEC_VALIDATION
        Setting X_CSI_SPEC_VALIDATION=true is the same as:
            X_CSI_SPEC_REQ_VALIDATION=true
            X_CSI_SPEC_REP_VALIDATION=true

    X_CSI_SPEC_REQ_VALIDATION
        A flag that enables the validation of CSI request messages.

    X_CSI_SPEC_REP_VALIDATION
        A flag that enables the validation of CSI response messages.
        Invalid responses are marshalled into a gRPC error with a code
        of "Internal."

    X_CSI_SPEC_DISABLE_LEN_CHECK
        A flag that disables validation of CSI message field lengths.

The flags -?,-h,-help may be used to print this screen.
`
