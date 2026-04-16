
package cmd

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
)

func newWebCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "web",
		Short: "Start the web dashboard",
		RunE:  runWeb,
	}
	cmd.Flags().Int("port", 7331, "Port to listen on")
	return cmd
}

func runWeb(cmd *cobra.Command, _ []string) error {
	port, _ := cmd.Flags().GetInt("port")
	addr := fmt.Sprintf(":%d", port)

	printErr(fmt.Sprintf("Starting web dashboard at http://localhost%s", addr))

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintln(w, "<html><body><h1>tddmaster dashboard</h1><p>Web UI not fully implemented in Go port.</p></body></html>")
	})
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		root, _ := resolveRoot()
		w.Header().Set("Content-Type", "application/json")
		if root == "" {
			fmt.Fprintln(w, `{"error":"not initialized"}`)
			return
		}
		fmt.Fprintf(w, `{"root":%q,"status":"ok"}`, root)
	})

	return http.ListenAndServe(addr, mux)
}
