// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package routes

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"

	"github.com/google/kf/pkg/kf/commands/completion"
	"github.com/google/kf/pkg/kf/commands/config"
	"github.com/google/kf/pkg/kf/commands/utils"
	"github.com/google/kf/pkg/kf/istio"
	"github.com/spf13/cobra"
)

// NewProxyRouteCommand creates a command capable of proxying a remote server locally.
func NewProxyRouteCommand(p *config.KfParams, ingressLister istio.IngressLister) *cobra.Command {
	var (
		gateway string
		port    int
		noStart bool
	)

	cmd := &cobra.Command{
		Use:     "proxy-route ROUTE",
		Short:   "Create a proxy to a route on a local port",
		Example: `kf proxy-route myhost.example.com`,
		Long: `
	This command creates a local proxy to a remote gateway modifying the request
	headers to make requests with the host set as the specified route.

	You can manually specify the gateway or have it autodetected based on your
	cluster.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := utils.ValidateNamespace(p); err != nil {
				return err
			}

			host := args[0]
			cmd.SilenceUsage = true

			if gateway == "" {
				fmt.Fprintln(cmd.OutOrStdout(), "Autodetecting app gateway. Specify a custom gateway using the --gateway flag.")

				ingress, err := istio.ExtractIngressFromList(ingressLister.ListIngresses())
				if err != nil {
					return err
				}
				gateway = ingress
			}

			listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
			if err != nil {
				return err
			}

			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "Forwarding requests from %s to %s with host %s\n", listener.Addr(), gateway, host)
			fmt.Fprintln(w, "Example GET:")
			fmt.Fprintf(w, "  curl %s\n", listener.Addr())
			fmt.Fprintf(w, "  (curl -H \"Host: %s\" http://%s)\n", host, gateway)
			fmt.Fprintln(w, "Example POST:")
			fmt.Fprintf(w, "  curl --request POST %s --data \"POST data\"\n", listener.Addr())
			fmt.Fprintf(w, "  (curl --request POST -H \"Host: %s\" http://%s --data \"POST data\")\n", host, gateway)
			fmt.Fprintln(w, "Browser link:")
			fmt.Fprintf(w, "  http://%s\n", listener.Addr())

			fmt.Fprintln(w)

			if noStart {
				fmt.Fprintln(cmd.OutOrStdout(), "exiting because no-start flag was provided")
				return nil
			}
			
			return http.Serve(listener, createProxy(cmd.OutOrStdout(), host, gateway))
		},
	}

	cmd.Flags().StringVar(
		&gateway,
		"gateway",
		"",
		"HTTP gateway to route requests to (default: autodetected from cluster)",
	)

	cmd.Flags().IntVar(
		&port,
		"port",
		8080,
		"Local port to listen on",
	)

	cmd.Flags().BoolVar(
		&noStart,
		"no-start",
		false,
		"Exit before starting the proxy",
	)
	cmd.Flags().MarkHidden("no-start")

	completion.MarkArgCompletionSupported(cmd, completion.AppCompletion)

	return cmd
}

func createProxy(w io.Writer, routeHost, gateway string) *httputil.ReverseProxy {
	logger := log.New(w, fmt.Sprintf("\033[34m[%s via %s]\033[0m ", routeHost, gateway), log.Ltime)

	return &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.Host = routeHost
			req.URL.Scheme = "http"
			req.URL.Host = gateway

			logger.Printf("%s %s\n", req.Method, req.URL.RequestURI())
		},
		ErrorLog: logger,
	}
}
