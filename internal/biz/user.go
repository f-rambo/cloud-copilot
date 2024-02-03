package biz

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/f-rambo/ocean/internal/conf"
	"github.com/f-rambo/ocean/utils"
	"github.com/go-kratos/kratos/v2/log"
	jwtv5 "github.com/golang-jwt/jwt/v5" // gorm customize data type
	"github.com/spf13/cast"
	"gorm.io/gorm"
)

type User struct {
	ID       int64  `json:"id,omitempty" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Name     string `json:"name,omitempty" gorm:"column:name; default:''; NOT NULL"`
	Email    string `json:"email,omitempty" gorm:"column:email; default:''; NOT NULL"`
	PassWord string `json:"password,omitempty" gorm:"column:password; default:''; NOT NULL"`
	State    int32  `json:"state,omitempty" gorm:"column:state; default:0; NOT NULL"`
	gorm.Model
}

const (
	AdminID   = -1
	AdminName = "admin"
)

type UserRepo interface {
	GetUserInfoByEmail(ctx context.Context, email string) (*User, error)
	GetUserByID(ctx context.Context, id int64) (*User, error)
	Save(ctx context.Context, user *User) error
}

type UserUseCase struct {
	repo     UserRepo
	log      *log.Helper
	authConf *conf.Auth
}

func NewUseUser(repo UserRepo, logger log.Logger, conf *conf.Auth) *UserUseCase {
	return &UserUseCase{repo: repo, log: log.NewHelper(logger), authConf: conf}
}

func (u *UserUseCase) SignIn(ctx context.Context, email string, password string) (token string, err error) {
	if email == u.authConf.Email && password == utils.Md5(u.authConf.PassWord) {
		return u.userToken(ctx, u.getAdmin())
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

func (u *UserUseCase) SignUp(ctx context.Context, email, name, password string) (token string, err error) {
	user := &User{
		Email:    email,
		Name:     name,
		PassWord: utils.Md5(password),
		State:    utils.Valid,
	}
	if err := u.repo.Save(ctx, user); err != nil {
		return "", err
	}
	return u.userToken(ctx, user)
}

// 通过token获取用户信息
func (u *UserUseCase) GetUserInfo(ctx context.Context) (*User, error) {
	token := ctx.Value(utils.TokenKey)
	if token == nil || cast.ToString(token) == "" {
		return nil, errors.New("token is null")
	}
	user, err := u.CheckJWT(ctx, cast.ToString(token))
	if err != nil {
		return nil, err
	}
	newToken, err := u.userToken(ctx, user)
	if err != nil {
		return nil, err
	}
	user.PassWord = newToken
	return user, nil
}

func (u *UserUseCase) GetUserByID(ctx context.Context, id int64) (*User, error) {
	if id == AdminID {
		return u.getAdmin(), nil
	}
	return u.repo.GetUserByID(ctx, id)
}

func (u *UserUseCase) getAdmin() *User {
	return &User{
		ID:    AdminID,
		Name:  AdminName,
		Email: u.authConf.Email,
		State: utils.Valid,
	}
}

// 解析jwt
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

// 获取jwt token
func (u *UserUseCase) userToken(ctx context.Context, user *User) (token string, err error) {
	claims := jwtv5.MapClaims{
		"id":    user.ID,
		"email": user.Email,
		"name":  user.Name,
		"state": user.State,
		"exp":   time.Now().Add(time.Hour * time.Duration(u.authConf.Exp)).Unix(),
	}
	return jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, claims).SignedString([]byte(u.authConf.Key))
}
