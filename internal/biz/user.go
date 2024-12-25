package biz

import (
	"context"

	"github.com/f-rambo/cloud-copilot/internal/conf"
	"github.com/f-rambo/cloud-copilot/utils"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/spf13/cast"
)

type UserData interface {
	GetUserInfoByEmail(ctx context.Context, email string) (*User, error)
	GetUserByID(ctx context.Context, id int64) (*User, error)
	Save(ctx context.Context, user *User) error
	GetUserByBatchID(ctx context.Context, ids []int64) ([]*User, error)
	GetUsers(ctx context.Context, username, email string, pageNum, pageSize int) (users []*User, total int64, err error)
	DeleteUser(ctx context.Context, id int64) error
	SignIn(context.Context, *User) error
	GetUserEmail(ctx context.Context, token string) (string, error)
}

type Thirdparty interface {
	GetUserEmail(ctx context.Context, token string) (string, error)
}

type UserAgent interface {
}

type UserUseCase struct {
	userData   UserData
	thirdparty Thirdparty
	log        *log.Helper
	conf       *conf.Bootstrap
}

func NewUseUser(userData UserData, thirdparty Thirdparty, logger log.Logger, conf *conf.Bootstrap) *UserUseCase {
	return &UserUseCase{userData: userData, thirdparty: thirdparty, log: log.NewHelper(logger), conf: conf}
}

func (u *UserUseCase) InitAdminUser(ctx context.Context) error {
	user := &User{
		Name:     "admin",
		Email:    u.conf.Auth.AdminEmail,
		Password: utils.Md5(u.conf.Auth.AdminPassword),
	}
	userData, err := u.userData.GetUserInfoByEmail(ctx, user.Email)
	if err != nil {
		return err
	}
	if userData != nil && userData.Id > 0 {
		user.Id = userData.Id
	}
	err = u.userData.Save(ctx, user)
	if err != nil {
		return err
	}
	return nil
}

func (u *UserUseCase) Save(ctx context.Context, user *User) error {
	return u.userData.Save(ctx, user)
}

func (u *UserUseCase) GetUsers(ctx context.Context, name, email string, pageNum, pageSize int) (users []*User, total int64, err error) {
	return u.userData.GetUsers(ctx, name, email, pageNum, pageSize)
}

func (u *UserUseCase) SignIn(ctx context.Context, user *User) error {
	if user.AccessToken != "" {
		email, err := u.thirdparty.GetUserEmail(ctx, user.AccessToken)
		if err != nil {
			return err
		}
		user.Email = email
		user.SignType = UserSignType_GITHUB
	} else {
		err := u.userData.SignIn(ctx, user)
		if err != nil {
			return err
		}
		user.SignType = UserSignType_CREDENTIALS
	}
	user.Status = UserStatus_USER_ENABLE
	err := u.Save(ctx, user)
	if err != nil {
		return err
	}
	return nil
}

func (u *UserUseCase) GetUserInfo(ctx context.Context) (*User, error) {
	userEmail := ctx.Value(UserEmailKey)
	return u.userData.GetUserInfoByEmail(ctx, cast.ToString(userEmail))
}

func (u *UserUseCase) GetUserByID(ctx context.Context, id int64) (*User, error) {
	return u.userData.GetUserByID(ctx, id)
}

func (u *UserUseCase) GetUserByBatchID(ctx context.Context, ids []int64) ([]*User, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	users, err := u.userData.GetUserByBatchID(ctx, ids)
	if err != nil {
		return nil, err
	}
	return users, nil
}

func (u *UserUseCase) DeleteUser(ctx context.Context, id int64) error {
	return u.userData.DeleteUser(ctx, id)
}
