package githubapi

import (
	"context"

	"github.com/google/go-github/v62/github"
	"github.com/pkg/errors"
)

type GithubApi struct {
	client      *github.Client
	accessToekn string
}

func NewClient(accessToekn string) *GithubApi {
	client := github.NewClient(nil).WithAuthToken(accessToekn)
	return &GithubApi{client: client, accessToekn: accessToekn}
}

func (g *GithubApi) GetCurrentUser(ctx context.Context) (*github.User, error) {
	user, res, err := g.client.Users.Get(ctx, "")
	if err != nil {
		return nil, err
	}
	if res.StatusCode != 200 {
		return nil, errors.New("failed to get current user")
	}
	return user, nil
}

// get repo releases
func (g *GithubApi) GetReleases(ctx context.Context, owner string, repo string) ([]*github.RepositoryRelease, error) {
	opts := &github.ListOptions{PerPage: 100}
	allReleases := make([]*github.RepositoryRelease, 0)
	for {
		releases, res, err := g.client.Repositories.ListReleases(ctx, owner, repo, opts)
		if err != nil {
			return nil, err
		}
		if res.StatusCode != 200 {
			return nil, errors.New("failed toget repo releases")
		}
		for _, release := range releases {
			if release.Prerelease != nil && *release.Prerelease {
				continue
			}
			allReleases = append(allReleases, release)
		}
		if res.NextPage == 0 {
			break
		}
		opts.Page = res.NextPage
	}
	return allReleases, nil
}
