// Package gh drives the GitHub side of a run over the REST API. Git plumbing on
// the workspace lives in package repo; this is only the hosted-forge operations.
package gh

import (
	"context"
	"fmt"

	"github.com/google/go-github/v74/github"
)

type Client struct {
	api   *github.Client
	owner string
	repo  string
}

func NewClient(token, owner, repo string) *Client {
	return &Client{
		api:   github.NewClient(nil).WithAuthToken(token),
		owner: owner,
		repo:  repo,
	}
}

func (c *Client) OpenPR(ctx context.Context, head, base, title, body string) (int, error) {
	pr, _, err := c.api.PullRequests.Create(ctx, c.owner, c.repo, &github.NewPullRequest{
		Title: &title,
		Head:  &head,
		Base:  &base,
		Body:  &body,
	})
	if err != nil {
		return 0, fmt.Errorf("open pr: %w", err)
	}
	return pr.GetNumber(), nil
}
