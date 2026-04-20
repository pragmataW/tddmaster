package service

import (
	"encoding/json"
	"os"
)

// pathExists reports whether a filesystem entry at path is accessible.
// A permission error is treated the same as not-existing; callers only
// care about presence, not about distinguishing error modes.
func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// readJSONMap reads path and decodes it as a JSON object. Returns nil
// on any I/O or decode failure, or if the top-level value is not an
// object.
func readJSONMap(path string) map[string]any {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil
	}
	return parsed
}

// readJSONField returns the named object field from a JSON file, or nil
// when the file is unreadable, the JSON is malformed, the field is
// missing, or the field value is not an object.
func readJSONField(path, field string) map[string]any {
	root := readJSONMap(path)
	if root == nil {
		return nil
	}
	val, ok := root[field].(map[string]any)
	if !ok {
		return nil
	}
	return val
}
