package biz

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"time"

	"github.com/f-rambo/cloud-copilot/internal/conf"
	"github.com/f-rambo/cloud-copilot/utils"
	"github.com/go-kratos/kratos/v2/log"
	jwtv5 "github.com/golang-jwt/jwt/v5"
	"github.com/spf13/cast"
)

const (
	AdminName ContextKey = "admin"

	TokenKey    ContextKey = "token"
	SignType    ContextKey = "sign_type"
	UserInfoKey ContextKey = "user_info"
	Exp         ContextKey = "exp"
	AuthKey     ContextKey = "auth"
)

type UserData interface {
	GetUserInfoByEmail(ctx context.Context, email string) (*User, error)
	GetUser(ctx context.Context, id int64) (*User, error)
	Save(ctx context.Context, user *User) error
	GetUserByBatchID(ctx context.Context, ids []int64) ([]*User, error)
	GetUsers(ctx context.Context, username, email string, pageNum, pageSize int) (users []*User, total int64, err error)
	DeleteUser(ctx context.Context, id int64) error
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
	os.Setenv(Exp.String(), string(conf.Auth.Exp))
	os.Setenv(AuthKey.String(), conf.Auth.Key)
	return &UserUseCase{userData: userData, thirdparty: thirdparty, log: log.NewHelper(logger), conf: conf}
}

func GetUserInfo(ctx context.Context) *User {
	v, ok := ctx.Value(UserInfoKey).(*User)
	if !ok {
		return nil
	}
	return v
}

func WithUser(ctx context.Context, u *User) context.Context {
	return context.WithValue(ctx, UserInfoKey, u)
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

func (u *UserUseCase) Register(ctx context.Context, user *User) error {
	user.SignType = UserSignType_CREDENTIALS
	if user.AccessToken != "" {
		email, err := u.thirdparty.GetUserEmail(ctx, user.AccessToken)
		if err != nil {
			return err
		}
		user.Email = email
		user.SignType = UserSignType_GITHUB
	}
	return u.Save(ctx, user)
}

func (u *UserUseCase) Enable(ctx context.Context, id int64) error {
	user, err := u.userData.GetUser(ctx, id)
	if err != nil {
		return err
	}
	user.Status = UserStatus_USER_ENABLE
	return u.Save(ctx, user)
}

func (u *UserUseCase) Disable(ctx context.Context, id int64) error {
	user, err := u.userData.GetUser(ctx, id)
	if err != nil {
		return err
	}
	user.Status = UserStatus_USER_DISABLE
	return u.Save(ctx, user)
}

func (u *UserUseCase) SignIn(ctx context.Context, user *User) (err error) {
	user.SignType = UserSignType_CREDENTIALS
	if user.Email == u.conf.Auth.AdminEmail && user.Password == utils.Md5(u.conf.Auth.AdminPassword) {
		user.AccessToken, err = GenerateJWT(user)
		if err != nil {
			return err
		}
		user.Status = UserStatus_USER_ENABLE
		return nil
	}
	if user.AccessToken != "" {
		email, err := u.thirdparty.GetUserEmail(ctx, user.AccessToken)
		if err != nil {
			return err
		}
		user.Email = email
		user.SignType = UserSignType_GITHUB
	}
	if user.Email == "" {
		return errors.New("email or password error")
	}
	if user.Status != UserStatus_USER_ENABLE {
		return errors.New("user is not enable")
	}
	user.AccessToken, err = GenerateJWT(user)
	if err != nil {
		return err
	}
	return nil
}

func (u *UserUseCase) GetUser(ctx context.Context, id int64) (*User, error) {
	return u.userData.GetUser(ctx, id)
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

func GenerateJWT(user *User) (token string, err error) {
	exp := os.Getenv(Exp.String())
	authKey := os.Getenv(AuthKey.String())
	userJsonString, err := json.Marshal(user)
	if err != nil {
		return "", err
	}
	claims := jwtv5.MapClaims{}
	err = json.Unmarshal(userJsonString, &claims)
	if err != nil {
		return "", err
	}
	claims["exp"] = time.Now().Add(time.Hour * time.Duration(cast.ToDuration(exp))).Unix()
	return jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, claims).SignedString([]byte(authKey))
}

func ValidateJWT(token string) (*User, error) {
	claims := jwtv5.MapClaims{}
	authKey := os.Getenv(AuthKey.String())
	_, err := jwtv5.ParseWithClaims(token, &claims, func(token *jwtv5.Token) (interface{}, error) {
		return []byte(authKey), nil
	})
	if err != nil {
		return nil, err
	}
	exp, ok := claims[Exp.String()]
	if !ok {
		return nil, errors.New("invalid expiration time")
	}
	if time.Now().Unix() > cast.ToInt64(exp) {
		return nil, errors.New("token is expired")
	}
	user := &User{}
	claimsJsonString, err := json.Marshal(claims)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(claimsJsonString, user)
	if err != nil {
		return nil, err
	}
	return user, nil
}
