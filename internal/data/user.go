package data

import (
	"context"

	"github.com/f-rambo/ocean/internal/biz"
	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
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
	user := &biz.User{}
	err := u.data.db.Where("email = ?", email).First(user).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return user, nil
}

func (u *UserRepo) Save(ctx context.Context, user *biz.User) error {
	return u.data.db.Save(user).Error
}

func (u *UserRepo) GetUserByID(ctx context.Context, id int64) (*biz.User, error) {
	user := &biz.User{}
	err := u.data.db.Where("id = ?", id).First(user).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return user, nil
}
