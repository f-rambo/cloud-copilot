package biz

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/f-rambo/ocean/internal/conf"
	"github.com/f-rambo/ocean/utils"
	"github.com/go-kratos/kratos/v2/log"
	jwtv5 "github.com/golang-jwt/jwt/v5"
	"github.com/spf13/cast"
)

type User struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	PassWord string `json:"password"`
}

type UserRepo interface {
	GetUserInfoByEmail(ctx context.Context, email string) (*User, error)
	Save(ctx context.Context, user *User) error
}

type UserUseCase struct {
	repo     UserRepo
	log      *log.Helper
	authConf conf.Auth
}

func NewUseUser(repo UserRepo, logger log.Logger) *UserUseCase {
	return &UserUseCase{repo: repo, log: log.NewHelper(logger)}
}

func (u *UserUseCase) SignIn(ctx context.Context, email string, password string) (token string, err error) {
	if email == u.authConf.Email && password == utils.Md5(u.authConf.PassWord) {
		return u.userToken(ctx, &User{
			ID:       1,
			Name:     "admin",
			Email:    u.authConf.Email,
			PassWord: u.authConf.PassWord,
		})
	}
	userInfo, err := u.repo.GetUserInfoByEmail(ctx, email)
	if err != nil {
		return
	}
	if userInfo.PassWord != password {
		return token, errors.New("password error")
	}
	return u.userToken(ctx, userInfo)
}

func (u *UserUseCase) SignOut(ctx context.Context) error {
	return nil
}

func (u *UserUseCase) GetUserInfo(ctx context.Context) (*User, error) {
	token := ctx.Value("token")
	if token == nil || cast.ToString(token) == "" {
		return nil, errors.New("token is null")
	}
	return u.CheckJWT(ctx, cast.ToString(token))
}

func (u *UserUseCase) CheckJWT(ctx context.Context, tokenString string) (*User, error) {
	token, err := jwtv5.Parse(tokenString, func(token *jwtv5.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwtv5.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(u.authConf.Key), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(jwtv5.MapClaims)
	if !ok || cast.ToInt64(claims["exp"]) < time.Now().Unix() {
		return nil, errors.New("invalid token")
	}
	return &User{
		ID:    cast.ToInt64(claims["id"]),
		Email: cast.ToString(claims["email"]),
		Name:  cast.ToString(claims["name"]),
	}, nil
}

func (u *UserUseCase) SignUp(ctx context.Context, email, name, password string) (token string, err error) {
	user := &User{
		Email:    email,
		Name:     name,
		PassWord: utils.Md5(password),
	}
	if err := u.repo.Save(ctx, user); err != nil {
		return "", err
	}
	return u.userToken(ctx, user)
}

func (u *UserUseCase) userToken(ctx context.Context, user *User) (token string, err error) {
	claims := jwtv5.MapClaims{
		"id":    user.ID,
		"email": user.Email,
		"name":  user.Name,
		"exp":   time.Now().Add(time.Hour * time.Duration(u.authConf.Exp)).Unix(),
	}
	return jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, claims).SignedString([]byte(u.authConf.Key))
}
