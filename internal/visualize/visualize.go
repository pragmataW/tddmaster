package visualize

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/pragmataW/tddmaster/internal/paths"
)

//go:embed dashboard.html
var DashboardHTML string

func GetHandler(root, slug string) (http.Handler, error) {
	specDir := paths.SpecDir(root, slug)

	dashboardDir := filepath.Join(specDir, "dashboard")
	if err := os.MkdirAll(dashboardDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create dashboard directory: %w", err)
	}

	htmlPath := filepath.Join(dashboardDir, "index.html")
	if err := os.WriteFile(htmlPath, []byte(DashboardHTML), 0o644); err != nil {
		return nil, fmt.Errorf("failed to write dashboard html: %w", err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/", "/dashboard", "/dashboard/", "/dashboard/index.html":
			http.ServeFile(w, r, htmlPath)
		default:
			http.NotFound(w, r)
		}
	})

	mux.HandleFunc("/progress.json", serveJSONFile(paths.SpecProgress(root, slug)))
	mux.HandleFunc("/settings.json", serveJSONFile(paths.SpecSettings(root, slug)))
	mux.HandleFunc("/state.json", serveJSONFile(paths.SpecState(root, slug)))

	mux.HandleFunc("/spec.md", func(w http.ResponseWriter, r *http.Request) {
		specPath := paths.SpecMd(root, slug)
		if _, err := os.Stat(specPath); err != nil {
			if os.IsNotExist(err) {
				http.NotFound(w, r)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/markdown")
		http.ServeFile(w, r, specPath)
	})

	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"hash": %q}`, CalculateHash(root, slug))
	})

	return mux, nil
}

func serveJSONFile(path string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				http.NotFound(w, r)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		var rm json.RawMessage
		if err := json.Unmarshal(data, &rm); err != nil {
			http.Error(w, "invalid json", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}
}

func ListenOnFreePort() (net.Listener, string, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, "", err
	}
	port := listener.Addr().(*net.TCPAddr).Port
	addr := fmt.Sprintf("http://127.0.0.1:%d", port)
	return listener, addr, nil
}

func CalculateHash(root, slug string) string {
	h := fnv.New64a()
	for _, p := range []string{
		paths.SpecProgress(root, slug),
		paths.SpecState(root, slug),
		paths.SpecSettings(root, slug),
		paths.SpecMd(root, slug),
	} {
		if info, err := os.Stat(p); err == nil {
			h.Write([]byte(fmt.Sprintf("%s-%d-%d", filepath.Base(p), info.ModTime().UnixNano(), info.Size())))
		}
	}
	return fmt.Sprintf("%x", h.Sum64())
}

func OpenBrowser(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	default:
		return exec.Command("xdg-open", url).Start()
	}
}
