package image

import (
	"encoding/json"
	"fmt"
	"time"
)

type Image struct {
	Name       string            `json:"name"`
	Tag        string            `json:"tag"`
	ID         string            `json:"id"`
	Layers     []Layer           `json:"layers"`
	Config     ImageConfig       `json:"config"`
	Size       int64             `json:"size"`
	CreatedAt  time.Time         `json:"created_at"`
	Labels     map[string]string `json:"labels"`
}

type Layer struct {
	Digest string `json:"digest"`
	Size   int64  `json:"size"`
	Path   string `json:"path"`
}

type ImageConfig struct {
	Entrypoint []string          `json:"entrypoint"`
	Cmd        []string          `json:"cmd"`
	Env        []string          `json:"env"`
	WorkingDir string            `json:"working_dir"`
	User       string            `json:"user"`
	Labels     map[string]string `json:"labels"`
}

type Manifest struct {
	SchemaVersion int               `json:"schemaVersion"`
	MediaType     string            `json:"mediaType"`
	Config        ManifestDescriptor `json:"config"`
	Layers        []ManifestDescriptor `json:"layers"`
}

type ManifestDescriptor struct {
	MediaType string `json:"mediaType"`
	Size      int64  `json:"size"`
	Digest    string `json:"digest"`
}

func ParseManifest(data []byte) (*Manifest, error) {
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("unmarshal manifest: %w", err)
	}
	return &manifest, nil
}