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

	"github.com/pragmataW/tddmaster/internal/state"
)

//go:embed dashboard.html
var DashboardHTML string

func GetHandler(root, slug string) (http.Handler, error) {
	specDir := getSpecDir(root, slug)

	dashboardDir := filepath.Join(specDir, "dashboard")
	if err := os.MkdirAll(dashboardDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create dashboard directory: %w", err)
	}

	htmlPath := filepath.Join(dashboardDir, "index.html")
	if err := os.WriteFile(htmlPath, []byte(DashboardHTML), 0644); err != nil {
		return nil, fmt.Errorf("failed to write dashboard html: %w", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || r.URL.Path == "/dashboard" || r.URL.Path == "/dashboard/" || r.URL.Path == "/dashboard/index.html" {
			http.ServeFile(w, r, htmlPath)
			return
		}
		http.NotFound(w, r)
	})

	mux.HandleFunc("/progress.json", func(w http.ResponseWriter, r *http.Request) {
		progressPath := filepath.Join(specDir, "progress.json")
		if _, err := os.Stat(progressPath); err != nil {
			if os.IsNotExist(err) {
				http.NotFound(w, r)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		data, err := os.ReadFile(progressPath)
		if err != nil {
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
	})

	mux.HandleFunc("/spec.md", func(w http.ResponseWriter, r *http.Request) {
		specPath := filepath.Join(specDir, "spec.md")
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

	mux.HandleFunc("/state.json", func(w http.ResponseWriter, r *http.Request) {
		path1 := filepath.Join(specDir, "state.json")
		path2 := filepath.Join(root, ".tddmaster", ".state", "specs", slug+".json")

		var statePath string
		if _, err := os.Stat(path1); err == nil {
			statePath = path1
		} else if !os.IsNotExist(err) {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		} else if _, err := os.Stat(path2); err == nil {
			statePath = path2
		} else if !os.IsNotExist(err) {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		} else {
			http.NotFound(w, r)
			return
		}

		data, err := os.ReadFile(statePath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		var rm json.RawMessage
		if err := json.Unmarshal(data, &rm); err != nil {
			http.Error(w, "invalid json", http.StatusInternalServerError)
			return
		}
		resolvedState, err := state.ResolveState(root, &slug)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		resp, err := json.MarshalIndent(resolvedState, "", "  ")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(resp)
	})

	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		hashStr := CalculateHash(root, slug)
		fmt.Fprintf(w, `{"hash": %q}`, hashStr)
	})

	return mux, nil
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
	specDir := getSpecDir(root, slug)

	// Add spec state representation
	if st, err := state.ResolveState(root, &slug); err == nil {
		if data, err := json.Marshal(st); err == nil {
			h.Write(data)
		}
	}

	// Add progress.json modification time & size
	progressPath := filepath.Join(specDir, "progress.json")
	if info, err := os.Stat(progressPath); err == nil {
		h.Write([]byte(fmt.Sprintf("%d-%d", info.ModTime().UnixNano(), info.Size())))
	}

	// Add spec.md modification time & size
	specPath := filepath.Join(specDir, "spec.md")
	if info, err := os.Stat(specPath); err == nil {
		h.Write([]byte(fmt.Sprintf("%d-%d", info.ModTime().UnixNano(), info.Size())))
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

func getSpecDir(root, slug string) string {
	return filepath.Join(root, ".tddmaster", "specs", slug)
}
