package github

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/willabides/octo-go"
	"github.com/willabides/octo-go/components"
	"github.com/willabides/octo-go/octotest"
)

func TestNewPRLabelFetcher(t *testing.T) {
	t.Run("has labels", func(t *testing.T) {
		ctx := context.Background()
		server := octotest.New()
		owner := "foo"
		repo := "bar"
		id := 12
		req := &octo.PullsGetReq{
			Owner:      owner,
			Repo:       repo,
			PullNumber: int64(id),
		}
		respBody := &octo.PullsGetResponseBody{
			Id: 12,
			Labels: []components.PullRequestLabelsItem{
				{Name: "label 1"},
				{Name: "label 2"},
				{Name: "label 3"},
			},
		}
		want := []string{"label 1", "label 2", "label 3"}
		server.Expect(req, octotest.JSONResponder(200, respBody))
		fetcher := NewPRLabelFetcher(ctx, owner, repo, server.Client()...)
		got, err := fetcher.FetchPRLabels(id)
		require.NoError(t, err)
		require.Equal(t, want, got)
	})

	t.Run("no labels", func(t *testing.T) {
		ctx := context.Background()
		server := octotest.New()
		owner := "foo"
		repo := "bar"
		id := 12
		req := &octo.PullsGetReq{
			Owner:      owner,
			Repo:       repo,
			PullNumber: int64(id),
		}
		respBody := &octo.PullsGetResponseBody{Id: 12}
		server.Expect(req, octotest.JSONResponder(200, respBody))
		fetcher := NewPRLabelFetcher(ctx, owner, repo, server.Client()...)
		got, err := fetcher.FetchPRLabels(id)
		require.NoError(t, err)
		require.Empty(t, got)
	})

	t.Run("not found", func(t *testing.T) {
		ctx := context.Background()
		server := octotest.New()
		owner := "foo"
		repo := "bar"
		id := 12
		req := &octo.PullsGetReq{
			Owner:      owner,
			Repo:       repo,
			PullNumber: int64(id),
		}
		server.Expect(req, octotest.JSONResponder(http.StatusNotFound, "not found"))
		fetcher := NewPRLabelFetcher(ctx, owner, repo, server.Client()...)
		got, err := fetcher.FetchPRLabels(id)
		require.Error(t, err)
		require.Empty(t, got)
	})
}
