package biz

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/f-rambo/ocean/internal/conf"
	"github.com/f-rambo/ocean/pkg/githubapi"
	"github.com/f-rambo/ocean/utils"
	"github.com/go-kratos/kratos/v2/log"
	jwtv5 "github.com/golang-jwt/jwt/v5" // gorm customize data type
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
	UserStateEnable  = "enable"
	UserStateDisable = "disable"
)

const (
	SignTypeGithub = "GITHUB"
	SignTypeBasic  = "CREDENTIALS"
)

const (
	AdminID   = -1
	AdminName = "admin"
)

const (
	TokenKey     = "token"
	SignType     = "sign_type"
	UserEmailKey = "user_email"
)

type UserRepo interface {
	GetUserInfoByEmail(ctx context.Context, email string) (*User, error)
	GetUserByID(ctx context.Context, id int64) (*User, error)
	Save(ctx context.Context, user *User) error
	GetUserByBatchID(ctx context.Context, ids []int64) ([]*User, error)
	GetUsers(ctx context.Context, username, email string, pageNum, pageSize int) (users []*User, total int64, err error)
	DeleteUser(ctx context.Context, id int64) error
}

type UserUseCase struct {
	repo     UserRepo
	log      *log.Helper
	authConf *conf.Auth
}

func NewUseUser(repo UserRepo, logger log.Logger, conf *conf.Bootstrap) *UserUseCase {
	cAuth := conf.GetOceanAuth()
	return &UserUseCase{repo: repo, log: log.NewHelper(logger), authConf: &cAuth}
}

// add update user 注册
func (u *UserUseCase) Save(ctx context.Context, user *User) error {
	if user.AccessToken != "" {
		if user.SignType == SignTypeGithub {
			githubUser, err := githubapi.NewClient(user.AccessToken).GetCurrentUser(ctx)
			if err != nil {
				return err
			}
			user.Name = *githubUser.Name
			user.Email = *githubUser.Email
		}
	}
	if user.Email == "" {
		return errors.New("email is null")
	}
	if user.ID == 0 {
		repoUser, err := u.repo.GetUserInfoByEmail(ctx, user.Email)
		if err != nil {
			return err
		}
		if repoUser != nil {
			user.ID = repoUser.ID
		} else {
			user.State = UserStateDisable
		}
	}
	if user.PassWord != "" {
		user.PassWord = utils.Md5(user.PassWord)
	}
	return u.repo.Save(ctx, user)
}

// 获取用户列表
func (u *UserUseCase) GetUsers(ctx context.Context, username, email string, pageNum, pageSize int) (users []*User, total int64, err error) {
	if username == AdminName {
		return []*User{u.getAdmin()}, 1, nil
	}
	users, total, err = u.repo.GetUsers(ctx, username, email, pageNum, pageSize)
	if err != nil {
		return nil, 0, err
	}
	for _, user := range users {
		if user.ID == AdminID {
			user.Name = AdminName
		}
	}
	return users, total, nil
}

func (u *UserUseCase) adminSignIn(user *User) (*User, bool, error) {
	if user.Email == u.authConf.Email && user.PassWord == utils.Md5(u.authConf.PassWord) {
		accessToken, err := u.encodeToken(u.getAdmin())
		if err != nil {
			return nil, true, err
		}
		return &User{
			ID:          AdminID,
			Name:        AdminName,
			Email:       u.authConf.Email,
			AccessToken: accessToken,
			SignType:    SignTypeBasic,
			State:       UserStateEnable,
		}, true, nil
	}
	return nil, false, nil
}

// thirdparty sign in
func (u *UserUseCase) thirdpartySignIn(ctx context.Context, user *User) (*User, bool, error) {
	if user.AccessToken != "" {
		switch strings.ToUpper(user.SignType) {
		case SignTypeGithub:
			githubUser, err := githubapi.NewClient(user.AccessToken).GetCurrentUser(ctx)
			if err != nil {
				return nil, true, err
			}
			if githubUser == nil {
				return nil, true, errors.New("github user is null")
			}
		default:
			return nil, true, errors.New("sign type error")
		}
		userInfo, err := u.repo.GetUserInfoByEmail(ctx, user.Email)
		if err != nil {
			return nil, true, err
		}
		user.ID = userInfo.ID
		user.State = UserStateEnable
		if user.ID == 0 {
			user.State = UserStateDisable
		}
		err = u.repo.Save(ctx, user)
		if err != nil {
			return nil, true, err
		}
		return user, true, nil
	}
	return nil, false, nil
}

func (u *UserUseCase) SignIn(ctx context.Context, user *User) (*User, error) {
	data, ok, err := u.adminSignIn(user)
	if ok {
		return data, err
	}
	data, ok, err = u.thirdpartySignIn(ctx, user)
	if ok {
		return data, err
	}
	userInfo, err := u.repo.GetUserInfoByEmail(ctx, user.Email)
	if err != nil {
		return nil, err
	}
	if userInfo == nil || userInfo.ID == 0 {
		return nil, errors.New("user not exist")
	}
	if userInfo.PassWord != user.PassWord {
		return nil, errors.New("password error")
	}
	userInfo.AccessToken, err = u.encodeToken(userInfo)
	if err != nil {
		return nil, err
	}
	userInfo.SignType = SignTypeBasic
	err = u.repo.Save(ctx, userInfo)
	if err != nil {
		return nil, err
	}
	return userInfo, nil
}

// 通过token获取用户信息
func (u *UserUseCase) GetUserInfo(ctx context.Context) (*User, error) {
	token := ctx.Value(TokenKey)
	signType := ctx.Value(SignType)
	userEmail := ctx.Value(UserEmailKey)
	if token == nil || cast.ToString(token) == "" {
		return nil, errors.New("token is null")
	}
	if signType == nil || cast.ToString(signType) == "" {
		return nil, errors.New("sign type is null")
	}
	user := &User{}
	if strings.ToUpper(cast.ToString(signType)) == SignTypeGithub {
		githubUser, err := githubapi.NewClient(cast.ToString(token)).GetCurrentUser(ctx)
		if err != nil {
			return nil, err
		}
		if githubUser == nil {
			return nil, errors.New("github user is null")
		}
		user.Name = *githubUser.Name
		user.Email = cast.ToString(userEmail)
	}
	if strings.ToUpper(cast.ToString(signType)) == SignTypeBasic {
		userJwt, err := u.DecodeToken(ctx, cast.ToString(token))
		if err != nil {
			return nil, err
		}
		user.Name = userJwt.Name
		user.Email = userJwt.Email
	}
	if user.Email == "" {
		return nil, errors.New("email is null")
	}
	return u.repo.GetUserInfoByEmail(ctx, user.Email)
}

func (u *UserUseCase) GetUserByID(ctx context.Context, id int64) (*User, error) {
	if id == AdminID {
		return u.getAdmin(), nil
	}
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
	for _, id := range ids {
		if id == AdminID {
			users = append(users, u.getAdmin())
			break
		}
	}
	return users, nil
}

func (u *UserUseCase) DeleteUser(ctx context.Context, id int64) error {
	if id == AdminID {
		return errors.New("admin can't be deleted")
	}
	return u.repo.DeleteUser(ctx, id)
}

func (u *UserUseCase) getAdmin() *User {
	return &User{
		ID:    AdminID,
		Name:  AdminName,
		Email: u.authConf.Email,
	}
}

func (u *UserUseCase) DecodeToken(ctx context.Context, t string) (*User, error) {
	token, err := jwtv5.Parse(t, func(token *jwtv5.Token) (interface{}, error) {
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

func (u *UserUseCase) encodeToken(user *User) (token string, err error) {
	claims := jwtv5.MapClaims{
		"id":    user.ID,
		"email": user.Email,
		"name":  user.Name,
		"state": user.State,
		"exp":   time.Now().Add(time.Hour * time.Duration(u.authConf.Exp)).Unix(),
	}
	return jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, claims).SignedString([]byte(u.authConf.Key))
}
