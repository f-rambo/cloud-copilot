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
