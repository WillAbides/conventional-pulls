package github

import (
	"context"

	"github.com/willabides/conventionalpulls"
	"github.com/willabides/octo-go"
)

type prLabelFetcher struct {
	client octo.Client
	owner  string
	repo   string
	getCtx func() context.Context
}

func (f *prLabelFetcher) FetchPRLabels(id int) ([]string, error) {
	client := f.client
	pull, err := client.PullsGet(f.getCtx(), &octo.PullsGetReq{
		Owner:      f.owner,
		Repo:       f.repo,
		PullNumber: int64(id),
	})
	if err != nil {
		return nil, err
	}
	labels := make([]string, len(pull.Data.Labels))
	for i, label := range pull.Data.Labels {
		labels[i] = label.Name
	}
	return labels, nil
}

// NewPRLabelFetcher returns a PRLabelFetcher that queries GitHub for PR Labels
func NewPRLabelFetcher(ctx context.Context, owner, repo string, opt ...octo.RequestOption) conventionalpulls.PRLabelFetcher {
	return &prLabelFetcher{
		client: opt,
		owner:  owner,
		repo:   repo,
		getCtx: func() context.Context {
			return ctx
		},
	}
}
