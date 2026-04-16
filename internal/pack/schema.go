
// Package pack provides types for pack manifests, registries, and installed packs.
package pack

import (
	"errors"

	"github.com/pragmataW/tddmaster/internal/state"
)

// =============================================================================
// Pack Manifest (pack.json)
// =============================================================================

type PackManifest struct {
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Description string            `json:"description"`
	Author      *string           `json:"author,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Requires    []string          `json:"requires,omitempty"`
	Rules       []string          `json:"rules,omitempty"`
	Concerns    []string          `json:"concerns,omitempty"`
	FolderRules map[string]string `json:"folderRules,omitempty"`
}

// =============================================================================
// Pack Registry (remote registry.json)
// =============================================================================

type PackRegistryEntry struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Version     string  `json:"version"`
	Tags        []string `json:"tags,omitempty"`
	Source      string  `json:"source"` // "builtin" | "local" | "remote"
	Specifier   *string `json:"specifier,omitempty"`
}

type PackRegistry struct {
	Packs []PackRegistryEntry `json:"packs"`
}

// =============================================================================
// Installed Pack (tracked in .tddmaster/packs.json)
// =============================================================================

type InstalledPack struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	InstalledAt string   `json:"installedAt"`
	Source      string   `json:"source"`
	Rules       []string `json:"rules"`
	Concerns    []string `json:"concerns"`
	FolderRules []string `json:"folderRules"`
}

type InstalledPacksFile struct {
	Installed []InstalledPack `json:"installed"`
}

// =============================================================================
// Built-in Pack (runtime representation with embedded content)
// =============================================================================

type BuiltinPack struct {
	Manifest            PackManifest                `json:"manifest"`
	RuleContents        map[string]string           `json:"ruleContents"`
	ConcernContents     []state.ConcernDefinition   `json:"concernContents"`
	FolderRuleContents  map[string]string           `json:"folderRuleContents,omitempty"`
}

// =============================================================================
// Validation
// =============================================================================

// ValidatePackManifest validates and returns the pack manifest from a raw
// decoded value. Returns an error if required fields are missing or empty.
func ValidatePackManifest(data interface{}) (*PackManifest, error) {
	obj, ok := data.(map[string]interface{})
	if !ok {
		return nil, errors.New("pack manifest must be an object")
	}

	name, _ := obj["name"].(string)
	if name == "" {
		return nil, errors.New("Pack manifest must have a non-empty 'name' field")
	}

	version, _ := obj["version"].(string)
	if version == "" {
		return nil, errors.New("Pack manifest must have a non-empty 'version' field")
	}

	description, _ := obj["description"].(string)
	if description == "" {
		return nil, errors.New("Pack manifest must have a non-empty 'description' field")
	}

	manifest := &PackManifest{
		Name:        name,
		Version:     version,
		Description: description,
	}

	if author, ok := obj["author"].(string); ok {
		manifest.Author = &author
	}

	if tags, ok := obj["tags"].([]interface{}); ok {
		for _, t := range tags {
			if s, ok := t.(string); ok {
				manifest.Tags = append(manifest.Tags, s)
			}
		}
	}

	if requires, ok := obj["requires"].([]interface{}); ok {
		for _, r := range requires {
			if s, ok := r.(string); ok {
				manifest.Requires = append(manifest.Requires, s)
			}
		}
	}

	if rules, ok := obj["rules"].([]interface{}); ok {
		for _, r := range rules {
			if s, ok := r.(string); ok {
				manifest.Rules = append(manifest.Rules, s)
			}
		}
	}

	if concerns, ok := obj["concerns"].([]interface{}); ok {
		for _, c := range concerns {
			if s, ok := c.(string); ok {
				manifest.Concerns = append(manifest.Concerns, s)
			}
		}
	}

	if folderRules, ok := obj["folderRules"].(map[string]interface{}); ok {
		manifest.FolderRules = make(map[string]string)
		for k, v := range folderRules {
			if s, ok := v.(string); ok {
				manifest.FolderRules[k] = s
			}
		}
	}

	return manifest, nil
}

// CreateEmptyPacksFile returns an InstalledPacksFile with no installed packs.
func CreateEmptyPacksFile() InstalledPacksFile {
	return InstalledPacksFile{
		Installed: []InstalledPack{},
	}
}
