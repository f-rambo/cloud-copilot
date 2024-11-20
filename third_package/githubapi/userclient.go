package githubapi

import (
	"context"

	"github.com/f-rambo/cloud-copilot/internal/biz"
	"github.com/f-rambo/cloud-copilot/internal/conf"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/pkg/errors"
)

type UserClient struct {
	c   *conf.Bootstrap
	log *log.Helper
}

func NewUserClient(c *conf.Bootstrap, logger log.Logger) biz.Thirdparty {
	return &UserClient{
		c:   c,
		log: log.NewHelper(logger),
	}
}

func (u *UserClient) GetUserEmail(ctx context.Context, token string) (string, error) {
	githubUser, err := NewClient(token).GetCurrentUser(ctx)
	if err != nil {
		return "", err
	}
	if githubUser == nil {
		return "", errors.New("github user is null")
	}
	return githubUser.GetEmail(), nil
}
