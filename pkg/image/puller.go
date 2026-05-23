package image

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Yancy/YContainer/internal/utils"
)

const (
	DockerHubRegistry = "registry-1.docker.io"
	DockerHubAuth     = "auth.docker.io"
	YcImagesDir       = "/var/lib/yc/images"
)

type Puller struct {
	registry string
	client   *http.Client
	logger   *utils.Logger
}

func NewPuller(logger *utils.Logger) *Puller {
	return &Puller{
		registry: DockerHubRegistry,
		client:   &http.Client{},
		logger:   logger,
	}
}

func (p *Puller) Pull(imageRef string) (*Image, error) {
	repo, tag := parseImageRef(imageRef)
	if tag == "" {
		tag = "latest"
	}

	p.logger.Info("Pulling image %s:%s from %s", repo, tag, p.registry)

	token, err := p.getToken(repo)
	if err != nil {
		return nil, fmt.Errorf("get auth token: %w", err)
	}

	manifest, err := p.getManifest(repo, tag, token)
	if err != nil {
		return nil, fmt.Errorf("get manifest: %w", err)
	}

	imageDir := filepath.Join(YcImagesDir, repo, tag)
	if err := os.MkdirAll(imageDir, 0755); err != nil {
		return nil, fmt.Errorf("create image dir: %w", err)
	}

	manifestData, _ := json.MarshalIndent(manifest, "", "  ")
	os.WriteFile(filepath.Join(imageDir, "manifest.json"), manifestData, 0644)

	configData, err := p.downloadBlob(repo, manifest.Config.Digest, token)
	if err != nil {
		return nil, fmt.Errorf("download config: %w", err)
	}
	os.WriteFile(filepath.Join(imageDir, "config.json"), configData, 0644)

	var layers []Layer
	for i, layer := range manifest.Layers {
		p.logger.Info("Downloading layer %d/%d: %s", i+1, len(manifest.Layers), layer.Digest[:20])

		layerDir := filepath.Join(imageDir, "layers", layer.Digest)
		if err := os.MkdirAll(layerDir, 0755); err != nil {
			return nil, fmt.Errorf("create layer dir: %w", err)
		}

		tarPath := filepath.Join(layerDir, "layer.tar")
		data, err := p.downloadBlob(repo, layer.Digest, token)
		if err != nil {
			return nil, fmt.Errorf("download layer %d: %w", i, err)
		}
		os.WriteFile(tarPath, data, 0644)

		extractDir := filepath.Join(layerDir, "rootfs")
		os.MkdirAll(extractDir, 0755)
		if err := extractTar(tarPath, extractDir); err != nil {
			return nil, fmt.Errorf("extract layer %d: %w", i, err)
		}

		layers = append(layers, Layer{
			Digest: layer.Digest,
			Size:   layer.Size,
			Path:   extractDir,
		})
	}

	var config ImageConfig
	json.Unmarshal(configData, &config)

	image := &Image{
		Name:      repo,
		Tag:       tag,
		ID:        manifest.Config.Digest,
		Layers:    layers,
		Config:    config,
		Size:      calculateTotalSize(layers),
	}

	p.logger.Info("Successfully pulled image %s:%s (%d layers)", repo, tag, len(layers))
	return image, nil
}

func (p *Puller) getToken(repo string) (string, error) {
	url := fmt.Sprintf("https://%s/token?service=registry.docker.io&scope=repository:%s:pull",
		DockerHubAuth, repo)

	resp, err := p.client.Get(url)
	if err != nil {
		return "", fmt.Errorf("request token: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode token: %w", err)
	}

	return result.Token, nil
}

func (p *Puller) getManifest(repo, tag, token string) (*Manifest, error) {
	url := fmt.Sprintf("https://%s/v2/%s/manifests/%s", p.registry, repo, tag)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request manifest: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}

	return ParseManifest(data)
}

func (p *Puller) downloadBlob(repo, digest, token string) ([]byte, error) {
	url := fmt.Sprintf("https://%s/v2/%s/blobs/%s", p.registry, repo, digest)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request blob: %w", err)
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

func parseImageRef(ref string) (repo, tag string) {
	if strings.Contains(ref, ":") {
		parts := strings.Split(ref, ":")
		return parts[0], parts[1]
	}
	return ref, "latest"
}

func calculateTotalSize(layers []Layer) int64 {
	var total int64
	for _, l := range layers {
		total += l.Size
	}
	return total
}

func extractTar(tarPath, destDir string) error {
	return execCommand("tar", "-xf", tarPath, "-C", destDir)
}

func execCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	return cmd.Run()
}