package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Manifest struct {
	Checkpoints ManifestCheckpoints `json:"checkpoints"`
}

type ManifestCheckpoints struct {
	KeepLast int `json:"keep_last"`
	KeepDays int `json:"keep_days"`
}

func LoadManifest(path string) (Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, err
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return Manifest{}, fmt.Errorf("parse manifest: %w", err)
	}
	return m, nil
}

func DefaultManifestPath(repoRoot string) string {
	return filepath.Join(repoRoot, ".agit-manifest.json")
}
