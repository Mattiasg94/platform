package repo

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"
)

// Auth's zero value is an unauthenticated public clone (today's default); it is
// the seam for a GitHub App token later — see platform docs/to-do.md.
type Auth struct {
	Token string
}

func Checkout(ctx context.Context, repoURL, ref string, auth Auth) (dir string, cleanup func(), err error) {
	dir, err = os.MkdirTemp("", "workspace-*")
	if err != nil {
		return "", nil, fmt.Errorf("create workspace dir: %w", err)
	}
	cleanup = func() { _ = os.RemoveAll(dir) }

	cloneURL, err := authenticatedURL(repoURL, auth)
	if err != nil {
		cleanup()
		return "", nil, err
	}
	if err := cloneRef(ctx, cloneURL, ref, dir); err != nil {
		cleanup()
		return "", nil, err
	}
	return dir, cleanup, nil
}

func cloneRef(ctx context.Context, cloneURL, ref, dir string) error {
	// init+fetch+checkout, not `git clone --branch`, so ref can be a branch,
	// tag, or SHA uniformly.
	steps := [][]string{
		{"init", "-q"},
		{"remote", "add", "origin", cloneURL},
		{"fetch", "-q", "--depth", "1", "origin", ref},
		{"checkout", "-q", "FETCH_HEAD"},
	}
	for _, step := range steps {
		if err := git(ctx, dir, step...); err != nil {
			return err
		}
	}
	return nil
}

func CreateBranch(ctx context.Context, dir, branch string) error {
	return git(ctx, dir, "switch", "-c", branch)
}

func CommitAll(ctx context.Context, dir, message string) error {
	if err := git(ctx, dir, "add", "-A"); err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, "git", "-C", dir, "commit", "-q", "-m", message)
	// The ephemeral clone has no configured identity; supply a bot one so the
	// commit doesn't fail or borrow the host user's.
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=orchestrator", "GIT_AUTHOR_EMAIL=orchestrator@platform.local",
		"GIT_COMMITTER_NAME=orchestrator", "GIT_COMMITTER_EMAIL=orchestrator@platform.local",
	)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git commit: %w", err)
	}
	return nil
}

func Push(ctx context.Context, dir, branch string) error {
	return git(ctx, dir, "push", "-q", "origin", "HEAD:refs/heads/"+branch)
}

func HeadSHA(ctx context.Context, dir string) (string, error) {
	out, err := gitOutput(ctx, dir, "rev-parse", "HEAD")
	return strings.TrimSpace(out), err
}

func git(ctx context.Context, dir string, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...)
	// git's chatter must not reach our stdout, which carries the pod result JSON.
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		// Name only the subcommand — args may carry the tokenized remote URL.
		return fmt.Errorf("git %s: %w", args[0], err)
	}
	return nil
}

func gitOutput(ctx context.Context, dir string, args ...string) (string, error) {
	var out bytes.Buffer
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...)
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %w", args[0], err)
	}
	return out.String(), nil
}

func authenticatedURL(repoURL string, auth Auth) (string, error) {
	if auth.Token == "" {
		return repoURL, nil
	}
	u, err := url.Parse(repoURL)
	if err != nil {
		return "", fmt.Errorf("parse repo url: %w", err)
	}
	if u.Scheme != "https" {
		return "", fmt.Errorf("token auth needs an https url, got scheme %q", u.Scheme)
	}
	u.User = url.UserPassword("x-access-token", auth.Token)
	return u.String(), nil
}
