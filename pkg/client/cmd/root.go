package cmd

import (
	"fmt"
	"net/url"
	"os"

	"github.com/cjdenio/underpass/pkg/client/tunnel"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var host string
var insecure bool
var port int
var subdomain string

var rootCmd = &cobra.Command{
	Use:   "upass",
	Short: "The Underpass CLI",
	Run: func(cmd *cobra.Command, args []string) {
		scheme := "wss"
		if insecure {
			scheme = "ws"
		}

		query := url.Values{}
		if subdomain != "" {
			query.Set("subdomain", subdomain)
		}

		u := url.URL{
			Scheme:   scheme,
			Path:     "start",
			Host:     host,
			RawQuery: query.Encode(),
		}

		t, err := tunnel.Connect(u.String(), fmt.Sprintf("http://localhost:%d", port))
		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Print("Started tunnel: ")
		if insecure {
			color.New(color.Bold, color.FgGreen).Printf("http://%s.%s", t.Subdomain, host)
		} else {
			color.New(color.Bold, color.FgGreen).Printf("https://%s.%s", t.Subdomain, host)
		}
		color.New(color.FgHiBlack).Print(" --> ")
		color.New(color.Bold, color.FgCyan).Printf("http://localhost:%d\n\n", port)

		if err = t.Wait(); err != nil {
			fmt.Printf("\n❌ Disconnected from server. %s\n", color.New(color.FgHiBlack).Sprint(err))
			os.Exit(1)
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringVar(&host, "host", "upass.clb.li", "Host to connect to")
	rootCmd.Flags().BoolVar(&insecure, "insecure", false, "[ADVANCED] don't tunnel over TLS")
	rootCmd.Flags().IntVarP(&port, "port", "p", 0, "Port to tunnel to")
	rootCmd.Flags().StringVarP(&subdomain, "subdomain", "s", "", "Request a custom subdomain")

	rootCmd.MarkFlagRequired("port")
	rootCmd.Flags().MarkHidden("insecure")
}
