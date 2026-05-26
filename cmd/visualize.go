package cmd

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

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

func getSpecDir(root, slug string) string {
	return filepath.Join(root, ".tddmaster", "specs", slug)
}

func newVisualizeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "visualize <slug>",
		Short: "Visualize spec and progress in a real-time updating web dashboard",
		Args:  cobra.ExactArgs(1),
		RunE:  runVisualize,
	}
	return cmd
}

func runVisualize(cmd *cobra.Command, args []string) error {
	slug := args[0]
	root, err := resolveRoot()
	if err != nil {
		return err
	}

	specDir := getSpecDir(root, slug)
	if _, err := os.Stat(specDir); err != nil {
		return fmt.Errorf("spec directory not found for slug %q: make sure the slug is correct and exists in .tddmaster/specs/", slug)
	}

	// Listen on a random open port
	listener, url, err := listenOnFreePort()
	if err != nil {
		return fmt.Errorf("failed to listen on an available port: %w", err)
	}

	handler, err := getVisualizeHandler(root, slug)
	if err != nil {
		return err
	}

	server := &http.Server{
		Handler: handler,
	}

	htmlPath := filepath.Join(specDir, "dashboard", "index.html")
	printErr(fmt.Sprintf("Dashboard template generated successfully under:\n  %s\n", htmlPath))
	printErr(fmt.Sprintf("Starting local dashboard web server at %s ...", url))

	// Open browser in the background
	go func() {
		// Wait a split second for server to startup
		time.Sleep(200 * time.Millisecond)
		printErr("Opening browser...")
		if err := visualize.OpenBrowser(url); err != nil {
			printErr(fmt.Sprintf("Failed to open browser automatically: %v\nYou can manually open it at %s", err, url))
		}
	}()

	// Start server (this blocks)
	if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("web server failed: %w", err)
	}

	return nil
}
