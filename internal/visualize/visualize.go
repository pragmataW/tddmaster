package visualize

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"html"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/pragmataW/tddmaster/internal/errs"
	"github.com/pragmataW/tddmaster/internal/paths"
	"github.com/pragmataW/tddmaster/internal/spec"
)

//go:embed dashboard.html
var DashboardHTML string

const (
	classCriteriaGWT = "criteria-gwt"
	classGwtTable    = "gwt-table"
	classGwtID       = "gwt-id"
)

func GetHandler(root, slug string) (http.Handler, error) {
	specDir := paths.SpecDir(root, slug)

	dashboardDir := filepath.Join(specDir, "dashboard")
	if err := os.MkdirAll(dashboardDir, 0o755); err != nil {
		return nil, errs.Wrap(errs.KeyCreateDashboardDir, err)
	}

	htmlPath := filepath.Join(dashboardDir, "index.html")
	// Criteria, istemci tarafında renderCriteria() ile yalnızca "Criteria"
	// sekmesindeki container-criteria'ya çizilir. Daha önce buraya server-side
	// enjekte edilen GWT bölümü footer'dan önce (tab container'larının dışında)
	// yer aldığı için her sekmede görünüyordu; o enjeksiyon kaldırıldı.
	if err := os.WriteFile(htmlPath, []byte(DashboardHTML), 0o644); err != nil {
		return nil, errs.Wrap(errs.KeyWriteDashboardHTML, err)
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
	mux.HandleFunc("/traceability.json", serveJSONFile(paths.SpecTraceability(root, slug)))

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

func writeGwtRow(b *strings.Builder, label, value string) {
	if strings.TrimSpace(value) == "" {
		return
	}
	b.WriteString(`<tr><th>`)
	b.WriteString(label)
	b.WriteString(`</th><td>`)
	b.WriteString(html.EscapeString(value))
	b.WriteString(`</td></tr>`)
}

func RenderCriteriaGWT(tasks []spec.Task) string {
	var b strings.Builder
	b.WriteString(`<div class="container-trace ` + classCriteriaGWT + `" id="container-criteria">`)
	for _, t := range tasks {
		if len(t.Criteria) == 0 {
			continue
		}
		b.WriteString(`<div class="panel trace-task-block">`)
		b.WriteString(`<div class="trace-task-title">`)
		b.WriteString(html.EscapeString(t.ID))
		if strings.TrimSpace(t.Title) != "" {
			b.WriteString(" · ")
			b.WriteString(html.EscapeString(t.Title))
		}
		b.WriteString(`</div>`)
		for _, c := range t.Criteria {
			b.WriteString(`<table class="` + classGwtTable + `" data-criterion="`)
			b.WriteString(html.EscapeString(c.ID))
			b.WriteString(`">`)
			b.WriteString(`<caption class="` + classGwtID + `">`)
			b.WriteString(html.EscapeString(strings.ToUpper(c.ID)))
			b.WriteString(`</caption>`)
			if strings.TrimSpace(c.Raw) != "" {
				writeGwtRow(&b, "RAW", c.Raw)
			} else {
				writeGwtRow(&b, "GIVEN", c.Given)
				writeGwtRow(&b, "WHEN", c.When)
				writeGwtRow(&b, "THEN", c.Then)
			}
			b.WriteString(`</table>`)
		}
		b.WriteString(`</div>`)
	}
	b.WriteString(`</div>`)
	return b.String()
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
		paths.SpecTraceability(root, slug),
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
