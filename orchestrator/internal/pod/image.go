package pod

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/oauth2/google"
)

const (
	registryHost = "us-central1-docker.pkg.dev"
	registryRepo = "platform" // infra/registry.tf
	imageName    = "agent"
)

// environmentFiles are the inputs that shape the image — and *only* those. The
// application source is deliberately absent: the env stage is source-free by
// construction (ADR-0009), so editing the project's code cannot change the image
// that runs it. Hashing the source instead would change the tag on every run,
// miss the cache every time, and rebuild on every execution — strictly worse than
// building blindly.
func (b *Builder) environmentFiles() []string {
	return []string{
		filepath.Join(b.projectDir, "Dockerfile"),    // the project's toolchain
		filepath.Join(agentHostPath(), "Dockerfile"), // the harness layered on it
	}
}

// imageRef names the image by the content of the files that define it. The tag is
// the identity of the environment: same inputs, same tag, and the image already
// exists. Change a Dockerfile and the tag moves, so the miss — and the rebuild —
// happen on their own, with nobody having to remember.
func (b *Builder) imageRef() (string, error) {
	sum := sha256.New()
	for _, path := range b.environmentFiles() {
		f, err := os.Open(path)
		if err != nil {
			return "", fmt.Errorf("hash environment: %w", err)
		}
		_, err = io.Copy(sum, f)
		_ = f.Close()
		if err != nil {
			return "", fmt.Errorf("hash %s: %w", path, err)
		}
	}
	tag := hex.EncodeToString(sum.Sum(nil))[:12]
	return fmt.Sprintf("%s/%s/%s/%s:%s", registryHost, b.gcpProject, registryRepo, imageName, tag), nil
}

// imageExists asks the registry whether this exact environment has been built
// before. Artifact Registry speaks the standard Docker Registry v2 API, so this is
// one authenticated HEAD — no pull, no build, no daemon.
func imageExists(ctx context.Context, ref string) (bool, error) {
	repository, tag, err := splitRef(ref)
	if err != nil {
		return false, err
	}

	token, err := registryToken(ctx)
	if err != nil {
		return false, err
	}

	url := fmt.Sprintf("https://%s/v2/%s/manifests/%s", registryHost, repository, tag)
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	// Without this the registry may answer 404 for an image that exists but is
	// stored under a manifest type we didn't say we accept.
	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json, "+
		"application/vnd.oci.image.manifest.v1+json, "+
		"application/vnd.oci.image.index.v1+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("query registry: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	default:
		return false, fmt.Errorf("registry answered %s for %s", resp.Status, ref)
	}
}

func registryToken(ctx context.Context) (string, error) {
	creds, err := google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return "", fmt.Errorf("find credentials: %w", err)
	}
	token, err := creds.TokenSource.Token()
	if err != nil {
		return "", fmt.Errorf("mint token: %w", err)
	}
	return token.AccessToken, nil
}

// splitRef turns host/project/repo/image:tag into (project/repo/image, tag) — the
// repository path the registry API wants, and the tag.
func splitRef(ref string) (repository, tag string, err error) {
	colon := strings.LastIndex(ref, ":")
	if colon < 0 {
		return "", "", fmt.Errorf("image ref %q has no tag", ref)
	}
	repository, found := strings.CutPrefix(ref[:colon], registryHost+"/")
	if !found {
		return "", "", fmt.Errorf("image ref %q is not in %s", ref, registryHost)
	}
	return repository, ref[colon+1:], nil
}
