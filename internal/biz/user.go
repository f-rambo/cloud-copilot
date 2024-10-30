package biz

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/spf13/cast"
	"gorm.io/gorm"
)

type User struct {
	ID          int64  `json:"id,omitempty" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Name        string `json:"name,omitempty" gorm:"column:name; default:''; NOT NULL"`
	Email       string `json:"email,omitempty" gorm:"column:email; default:''; NOT NULL"`
	PassWord    string `json:"password,omitempty" gorm:"column:password; default:''; NOT NULL"`
	State       string `json:"state,omitempty" gorm:"column:state; default:''; NOT NULL"`
	AccessToken string `json:"access_token,omitempty" gorm:"-"`
	SignType    string `json:"sign_type,omitempty" gorm:"column:sign_type; default:''; NOT NULL"`
	gorm.Model
}

const (
	AdminID       = -1
	AdminName     = "admin"
	AdminEmail    = "admin@admin.com"
	AdminPassword = "admin"
)

const (
	UserStateEnable  = "enable"
	UserStateDisable = "disable"
)

const (
	SignTypeGithub = "GITHUB"
	SignTypeBasic  = "CREDENTIALS"
)

type UserKey string

const (
	TokenKey     UserKey = "token"
	SignType     UserKey = "sign_type"
	UserEmailKey UserKey = "user_email"
)

type UserRepo interface {
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

type UserUseCase struct {
	repo       UserRepo
	thirdparty Thirdparty
	log        *log.Helper
}

func NewUseUser(repo UserRepo, thirdparty Thirdparty, logger log.Logger) *UserUseCase {
	return &UserUseCase{repo: repo, thirdparty: thirdparty, log: log.NewHelper(logger)}
}

func (u *UserUseCase) Save(ctx context.Context, user *User) error {
	return u.repo.Save(ctx, user)
}

func (u *UserUseCase) GetUsers(ctx context.Context, name, email string, pageNum, pageSize int) (users []*User, total int64, err error) {
	return u.repo.GetUsers(ctx, name, email, pageNum, pageSize)
}

func (u *UserUseCase) SignIn(ctx context.Context, user *User) error {
	if user.AccessToken != "" {
		fmt.Println(user.AccessToken)
		email, err := u.thirdparty.GetUserEmail(ctx, user.AccessToken)
		if err != nil {
			return err
		}
		user.Email = email
		user.SignType = SignTypeGithub
	} else {
		err := u.repo.SignIn(ctx, user)
		if err != nil {
			return err
		}
		user.SignType = SignTypeBasic
	}
	user.State = UserStateEnable
	if user.ID == AdminID {
		return nil
	}
	err := u.Save(ctx, user)
	if err != nil {
		return err
	}
	return nil
}

func (u *UserUseCase) GetUserInfo(ctx context.Context) (*User, error) {
	userEmail := ctx.Value(UserEmailKey)
	return u.repo.GetUserInfoByEmail(ctx, cast.ToString(userEmail))
}

func (u *UserUseCase) GetUserByID(ctx context.Context, id int64) (*User, error) {
	return u.repo.GetUserByID(ctx, id)
}

func (u *UserUseCase) GetUserByBatchID(ctx context.Context, ids []int64) ([]*User, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	users, err := u.repo.GetUserByBatchID(ctx, ids)
	if err != nil {
		return nil, err
	}
	return users, nil
}

func (u *UserUseCase) DeleteUser(ctx context.Context, id int64) error {
	return u.repo.DeleteUser(ctx, id)
}
