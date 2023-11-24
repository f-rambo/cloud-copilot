package data

import (
	"context"

	"github.com/f-rambo/ocean/internal/biz"
	"github.com/go-kratos/kratos/v2/log"
)

type UserRepo struct {
	data *Data
	log  *log.Helper
}

func NewUserRepo(data *Data, logger log.Logger) biz.UserRepo {
	return &UserRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

func (u *UserRepo) GetUserInfoByEmail(ctx context.Context, email string) (*biz.User, error) {
	return &biz.User{}, nil
}

func (u *UserRepo) Save(ctx context.Context, user *biz.User) error {
	return nil
}
