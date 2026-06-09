package spec

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/pragmataW/tddmaster/internal/paths"
)

const (
	jsonIndent = "  "
	dirPerm    = 0o755
	filePerm   = 0o644
)

func writeFile(dir, path string, data []byte) error {
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}
	if err := os.Chmod(tmpPath, filePerm); err != nil {
		os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return nil
}

func saveJSON(dir, path string, v any) error {
	data, err := json.MarshalIndent(v, "", jsonIndent)
	if err != nil {
		return err
	}
	return writeFile(dir, path, data)
}

func loadJSON[T any](path string) (T, error) {
	var v T
	data, err := os.ReadFile(path)
	if err != nil {
		return v, fmt.Errorf("read %s: %w", path, err)
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return v, fmt.Errorf("parse %s: %w", path, err)
	}
	return v, nil
}

func loadJSONOrEmpty[T any](path string) (T, error) {
	var zero T
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return zero, nil
	}
	return loadJSON[T](path)
}

func SaveState(root, slug string, s State) error {
	s.UpdatedAt = time.Now().UTC()
	return saveJSON(paths.SpecDir(root, slug), paths.SpecState(root, slug), s)
}

func LoadState(root, slug string) (State, error) {
	p := paths.SpecState(root, slug)
	s, err := loadJSON[State](p)
	if err != nil {
		return State{}, err
	}
	return s, nil
}

func SaveSettings(root, slug string, s Settings) error {
	return saveJSON(paths.SpecDir(root, slug), paths.SpecSettings(root, slug), s)
}

func LoadSettings(root, slug string) (Settings, error) {
	p := paths.SpecSettings(root, slug)
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return DefaultSettings(), nil
	}
	s, err := loadJSON[Settings](p)
	if err != nil {
		return Settings{}, err
	}
	return s, nil
}

func SaveProgress(root, slug string, p Progress) error {
	p.UpdatedAt = time.Now().UTC()
	return saveJSON(paths.SpecDir(root, slug), paths.SpecProgress(root, slug), p)
}

func LoadProgress(root, slug string) (Progress, error) {
	p := paths.SpecProgress(root, slug)
	pr, err := loadJSON[Progress](p)
	if err != nil {
		return Progress{}, err
	}
	return pr, nil
}

func SaveTraceability(root, slug string, t Traceability) error {
	return saveJSON(paths.SpecDir(root, slug), paths.SpecTraceability(root, slug), t)
}

func LoadTraceability(root, slug string) (Traceability, error) {
	p := paths.SpecTraceability(root, slug)
	tr, err := loadJSONOrEmpty[Traceability](p)
	if err != nil {
		return Traceability{}, err
	}
	if tr.Entries == nil {
		tr.Entries = map[string][]TraceEntry{}
	}
	if tr.Coverage == nil {
		tr.Coverage = map[string]float64{}
	}
	return tr, nil
}

func Exists(root, slug string) bool {
	_, err := os.Stat(paths.SpecState(root, slug))
	return err == nil
}

func SaveSpecMd(root, slug, content string) error {
	return writeFile(paths.SpecDir(root, slug), paths.SpecMd(root, slug), []byte(content))
}
