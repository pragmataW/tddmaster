package cmd

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/spf13/cobra"

	"github.com/pragmataW/tddmaster/internal/paths"
	"github.com/pragmataW/tddmaster/internal/spec"
	"github.com/pragmataW/tddmaster/internal/visualize"
)

var getVisualizeHandler func(root, slug string) (http.Handler, error)
var listenOnFreePort func() (net.Listener, string, error)
var dashboardHTML string

func init() {
	getVisualizeHandler = visualize.GetHandler
	listenOnFreePort = visualize.ListenOnFreePort
	dashboardHTML = visualize.DashboardHTML
}

func newVisualizeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "visualize <slug>",
		Short: "Visualize spec and progress in a real-time updating web dashboard",
		Args:  cobra.ExactArgs(1),
		RunE:  runVisualize,
	}
	addRootFlag(cmd)
	return cmd
}

func runVisualize(cmd *cobra.Command, args []string) error {
	slug := args[0]
	if !spec.ValidSlug(slug) {
		return fmt.Errorf("invalid slug %q", slug)
	}
	root, err := resolveRoot(cmd)
	if err != nil {
		return fmt.Errorf("resolve root: %w", err)
	}

	if !spec.Exists(root, slug) {
		return fmt.Errorf("spec directory not found for slug %q: make sure the slug is correct and exists in .tddmaster/specs/", slug)
	}

	listener, url, err := listenOnFreePort()
	if err != nil {
		return fmt.Errorf("failed to listen on an available port: %w", err)
	}

	handler, err := getVisualizeHandler(root, slug)
	if err != nil {
		return err
	}

	server := &http.Server{Handler: handler}

	htmlPath := paths.SpecDir(root, slug) + "/dashboard/index.html"
	out := cmd.ErrOrStderr()
	fmt.Fprintf(out, "Dashboard template generated under:\n  %s\n", htmlPath)
	fmt.Fprintf(out, "Starting local dashboard web server at %s ...\n", url)

	go func() {
		time.Sleep(200 * time.Millisecond)
		fmt.Fprintln(out, "Opening browser...")
		if err := visualize.OpenBrowser(url); err != nil {
			fmt.Fprintf(out, "Failed to open browser automatically: %v\nOpen it manually at %s\n", err, url)
		}
	}()

	if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("web server failed: %w", err)
	}

	return nil
}
