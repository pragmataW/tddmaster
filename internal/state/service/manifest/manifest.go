// Package manifest reads and writes the tddmaster config section inside
// .tddmaster/manifest.yml. YAML comments outside the tddmaster block are
// preserved through low-level node manipulation.
package manifest

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/pragmataW/tddmaster/internal/state/model"
	"github.com/pragmataW/tddmaster/internal/state/service/atomic"
	"github.com/pragmataW/tddmaster/internal/state/service/paths"
)

// ReadManifest reads the tddmaster section from manifest.yml.
func ReadManifest(root string) (*model.NosManifest, error) {
	filePath := filepath.Join(root, paths.ManifestFile)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, nil
	}

	var doc map[string]yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, nil
	}

	nosNode, ok := doc["tddmaster"]
	if !ok {
		return nil, nil
	}

	var manifest model.NosManifest
	if err := nosNode.Decode(&manifest); err != nil {
		return nil, nil
	}
	return &manifest, nil
}

// WriteManifest writes the tddmaster config to manifest.yml, preserving other keys.
func WriteManifest(root string, config model.NosManifest) error {
	dirPath := filepath.Join(root, paths.TddmasterDir)
	if err := os.MkdirAll(dirPath, 0o755); err != nil {
		return err
	}

	filePath := filepath.Join(root, paths.ManifestFile)

	var doc yaml.Node
	data, err := os.ReadFile(filePath)
	if err == nil {
		var rootNode yaml.Node
		if err := yaml.Unmarshal(data, &rootNode); err == nil && rootNode.Kind == yaml.DocumentNode {
			doc = *rootNode.Content[0]
		}
	}

	if doc.Kind == 0 {
		doc = yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	}

	configNode := &yaml.Node{}
	if err := configNode.Encode(config); err != nil {
		return err
	}
	if configNode.Kind == yaml.DocumentNode && len(configNode.Content) > 0 {
		configNode = configNode.Content[0]
	}
	configNode.HeadComment = " tddmaster orchestrator — inline comments in this section won't be preserved on next write"

	found := false
	for i := 0; i+1 < len(doc.Content); i += 2 {
		if doc.Content[i].Value == "tddmaster" {
			doc.Content[i+1] = configNode
			found = true
			break
		}
	}
	if !found {
		keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: "tddmaster", Tag: "!!str"}
		doc.Content = append(doc.Content, keyNode, configNode)
	}

	rootDoc := yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{&doc}}
	out, err := yaml.Marshal(&rootDoc)
	if err != nil {
		return err
	}
	return atomic.WriteFileAtomic(filePath, out, 0o644)
}

// IsInitialized checks if the project has been initialized (manifest.yml with tddmaster section).
func IsInitialized(root string) (bool, error) {
	filePath := filepath.Join(root, paths.ManifestFile)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return false, nil
	}

	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return false, nil
	}

	_, ok := doc["tddmaster"]
	return ok, nil
}

// CreateInitialManifest creates a new default NosManifest.
func CreateInitialManifest(
	concerns []string,
	tools []model.CodingToolId,
	project model.ProjectTraits,
) model.NosManifest {
	return model.NosManifest{
		Concerns:                   concerns,
		Tools:                      tools,
		Project:                    project,
		MaxIterationsBeforeRestart: 15,
		Tdd:                        &model.Manifest{TddMode: true, MaxVerificationRetries: 3, MaxRefactorRounds: 3},
		VerifyCommand:              nil,
		AllowGit:                   false,
		Command:                    "tddmaster",
	}
}

// ParseManifest deserializes a Manifest from raw YAML bytes.
// Unknown keys are ignored; missing keys receive their zero values.
func ParseManifest(data []byte) (model.Manifest, error) {
	var m model.Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return model.Manifest{}, err
	}
	return m, nil
}

// MarshalManifest serializes a Manifest to YAML bytes.
func MarshalManifest(m model.Manifest) ([]byte, error) {
	return yaml.Marshal(m)
}

// LoadManifest reads TDD-specific settings from <root>/.tddmaster/manifest.yml.
// It looks for the "tdd" key inside the "tddmaster" section; falls back to a
// top-level "tdd" key for backward compatibility. Returns a zero-value
// Manifest (no error) when no "tdd" key is present.
func LoadManifest(root string) (model.Manifest, error) {
	filePath := filepath.Join(root, paths.TddmasterDir, "manifest.yml")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return model.Manifest{}, err
	}

	var doc map[string]yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return model.Manifest{}, err
	}

	tddmasterNode, ok := doc["tddmaster"]
	if ok {
		var section map[string]yaml.Node
		if err := tddmasterNode.Decode(&section); err == nil {
			if tddNode, found := section["tdd"]; found {
				var m model.Manifest
				if err := tddNode.Decode(&m); err != nil {
					return model.Manifest{}, err
				}
				return m, nil
			}
		}
	}

	tddNode, ok := doc["tdd"]
	if !ok {
		return model.Manifest{}, nil
	}

	var m model.Manifest
	if err := tddNode.Decode(&m); err != nil {
		return model.Manifest{}, err
	}
	return m, nil
}
