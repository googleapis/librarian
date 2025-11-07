package automation

import (
	"context"
	"iter"
	"log/slog"

	"cloud.google.com/go/cloudbuild/apiv1/v2/cloudbuildpb"
	"github.com/google/go-github/v69/github"
	"github.com/googleapis/gax-go/v2"
)

type mockGitHubClient struct {
	prs []*github.PullRequest
	err error
}

func (m *mockGitHubClient) FindMergedPullRequestsWithPendingReleaseLabel(ctx context.Context, owner, repo string) ([]*github.PullRequest, error) {
	return m.prs, m.err
}

type mockCloudBuildClient struct {
	runError      error
	buildTriggers []*cloudbuildpb.BuildTrigger
	triggersRun   []string
}

func (c *mockCloudBuildClient) RunBuildTrigger(ctx context.Context, req *cloudbuildpb.RunBuildTriggerRequest, opts ...gax.CallOption) error {
	slog.Info("running fake RunBuildTrigger")
	if c.runError != nil {
		return c.runError
	}
	for _, t := range c.triggersRun {
		if t == req.TriggerId {
			return nil
		}
	}
	c.triggersRun = append(c.triggersRun, req.TriggerId)
	return nil
}

func (c *mockCloudBuildClient) ListBuildTriggers(ctx context.Context, req *cloudbuildpb.ListBuildTriggersRequest, opts ...gax.CallOption) iter.Seq2[*cloudbuildpb.BuildTrigger, error] {
	return func(yield func(*cloudbuildpb.BuildTrigger, error) bool) {
		for _, v := range c.buildTriggers {
			var err error
			if c.runError != nil {
				v = nil
				err = c.runError
			}
			if !yield(v, err) {
				return // Stop iteration if yield returns false
			}
		}
	}
}
