package data

import (
	"context"
	"fmt"
	"time"

	"github.com/f-rambo/cloud-copilot/internal/biz"
	"github.com/f-rambo/cloud-copilot/internal/conf"
	"github.com/go-kratos/kratos/v2/log"
	jwtv5 "github.com/golang-jwt/jwt/v5"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
	"gorm.io/gorm"
)

type UserRepo struct {
	data *Data
	log  *log.Helper
	c    *conf.Bootstrap
}

func NewUserRepo(data *Data, c *conf.Bootstrap, logger log.Logger) biz.UserData {
	return &UserRepo{
		data: data,
		c:    c,
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

func (u *UserRepo) GetUserByBatchID(ctx context.Context, ids []int64) ([]*biz.User, error) {
	users := make([]*biz.User, 0)
	err := u.data.db.Where("id in (?)", ids).Find(&users).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return users, nil
}

func (u *UserRepo) GetUsers(ctx context.Context, name, email string, pageNum, pageSize int) (users []*biz.User, total int64, err error) {
	users = make([]*biz.User, 0)
	db := u.data.db.Model(&biz.User{})
	if name != "" {
		db = db.Where("name LIKE ?", "%"+name+"%")
	}
	if email != "" {
		db = db.Where("email LIKE ?", "%"+email+"%")
	}
	err = db.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	err = db.Offset((pageNum - 1) * pageSize).Limit(pageSize).Find(&users).Error
	if err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

func (u *UserRepo) DeleteUser(ctx context.Context, id int64) error {
	return u.data.db.Delete(&biz.User{}, "id = ?", id).Error
}

func (u *UserRepo) SignIn(ctx context.Context, userParam *biz.User) error {
	user, err := u.GetUserInfoByEmail(ctx, userParam.Email)
	if err != nil {
		return err
	}
	if user == nil || user.Id == 0 {
		return errors.New("user not exist")
	}
	userParam.Id = user.Id
	if userParam.Password != user.Password {
		return errors.New("password error")
	}
	userParam.AccessToken, err = u.encodeToken(userParam)
	if err != nil {
		return err
	}
	return nil
}

func (u *UserRepo) encodeToken(user *biz.User) (token string, err error) {
	claims := jwtv5.MapClaims{
		"id":     user.Id,
		"email":  user.Email,
		"name":   user.Name,
		"status": user.Status,
		"exp":    time.Now().Add(time.Hour * time.Duration(u.c.Auth.Exp)).Unix(),
	}
	return jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, claims).SignedString([]byte(u.c.Auth.Key))
}

func (u *UserRepo) GetUserEmail(ctx context.Context, token string) (string, error) {
	user, err := u.decodeToken(ctx, token)
	if err != nil {
		return "", err
	}
	return user.Email, nil
}

func (u *UserRepo) decodeToken(_ context.Context, t string) (*biz.User, error) {
	token, err := jwtv5.Parse(t, func(token *jwtv5.Token) (any, error) {
		if _, ok := token.Method.(*jwtv5.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(u.c.Auth.Key), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(jwtv5.MapClaims)
	if !ok || cast.ToInt64(claims["exp"]) < time.Now().Unix() {
		return nil, errors.New("invalid token")
	}
	return &biz.User{
		Id:    cast.ToInt64(claims["id"]),
		Email: cast.ToString(claims["email"]),
		Name:  cast.ToString(claims["name"]),
	}, nil
}
