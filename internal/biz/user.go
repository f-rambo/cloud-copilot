package biz

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/f-rambo/cloud-copilot/internal/conf"
	"github.com/f-rambo/cloud-copilot/utils"
	"github.com/go-kratos/kratos/v2/log"
	jwtv5 "github.com/golang-jwt/jwt/v5"
	"github.com/spf13/cast"
)

// user error info
const (
	ErrUserNotFound      = "user not found"
	ErrUserAlreadyExists = "user already exists"
	ErrUserEmailNotFound = "user email not found"
	ErrUserPasswordError = "user password error"
)

const (
	AdminName ContextKey = "admin"

	TokenKey    ContextKey = "token"
	UserInfoKey ContextKey = "user_info"
)

type UserStatus int32

const (
	UserStatus_USER_INIT    UserStatus = 0
	UserStatus_USER_ENABLE  UserStatus = 1
	UserStatus_USER_DISABLE UserStatus = 2
	UserStatus_USER_DELETED UserStatus = 3
)

// UserStatus to string
func (s UserStatus) String() string {
	switch s {
	case UserStatus_USER_INIT:
		return "USER_INIT"
	case UserStatus_USER_ENABLE:
		return "USER_ENABLE"
	case UserStatus_USER_DISABLE:
		return "USER_DISABLE"
	case UserStatus_USER_DELETED:
		return "USER_DELETED"
	}
	return "UNKNOWN"
}

type User struct {
	Id                   int64      `json:"id,omitempty" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Name                 string     `json:"name,omitempty" gorm:"column:name;default:'';NOT NULL"`
	Namespace            string     `json:"namespace,omitempty" gorm:"column:namespace;default:'';NOT NULL"`
	Email                string     `json:"email,omitempty" gorm:"column:email;default:'';NOT NULL"`
	GitrepoName          string     `json:"gitrepo_name,omitempty" gorm:"column:gitrepo_name;default:'';NOT NULL"`
	ImagerepoName        string     `json:"imagerepo_name,omitempty" gorm:"column:imagerepo_name;default:'';NOT NULL"`
	Password             string     `json:"password,omitempty" gorm:"column:password;default:'';NOT NULL"`
	Status               UserStatus `json:"status,omitempty" gorm:"column:status;default:0;NOT NULL"`
	AccessToken          string     `json:"access_token,omitempty" gorm:"-"`
	WorkspaceId          int64      `json:"workspace_id,omitempty" gorm:"column:workspace_id;default:0;NOT NULL"`
	GitRepositoryToken   string     `json:"git_repository_token,omitempty" gorm:"column:gitrepository_token;default:'';NOT NULL"`
	ImageRepositoryToken string     `json:"image_repository_token,omitempty" gorm:"column:imagerepository_token;default:'';NOT NULL"`
}

type Role struct {
	Id          int64  `json:"id,omitempty" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Name        string `json:"name,omitempty" gorm:"column:name;default:'';NOT NULL"`
	Verbs       string `json:"verbs,omitempty" gorm:"column:verbs;default:'';NOT NULL"`         // GET,POST,PUT,DELETE
	Resources   string `json:"resources,omitempty" gorm:"column:resources;default:'';NOT NULL"` // project1:servicename1,servicename2|project2:servicename1  in the workspace
	Description string `json:"description,omitempty" gorm:"column:description;default:'';NOT NULL"`
}

type UserRole struct {
	Id     int64 `json:"id,omitempty" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	UserId int64 `json:"user_id,omitempty" gorm:"column:user_id;default:0;NOT NULL"`
	RoleId int64 `json:"role_id,omitempty" gorm:"column:role_id;default:0;NOT NULL"`
}

type UserData interface {
	GetUserInfoByEmail(ctx context.Context, email string) (*User, error)
	GetUser(ctx context.Context, id int64) (*User, error)
	Save(ctx context.Context, user *User) error
	GetUserByBatchID(ctx context.Context, ids []int64) ([]*User, error)
	GetUsers(ctx context.Context, username, email string, pageNum, pageSize int) (users []*User, total int64, err error)
	DeleteUser(ctx context.Context, id int64) error
}

type UserUseCase struct {
	userData UserData
	log      *log.Helper
	conf     *conf.Bootstrap
}

func NewUseUser(userData UserData, logger log.Logger, conf *conf.Bootstrap) *UserUseCase {
	return &UserUseCase{userData: userData, log: log.NewHelper(logger), conf: conf}
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

func (u *UserUseCase) GetUsers(ctx context.Context, name, email string, pageNum, pageSize int) (users []*User, total int64, err error) {
	return u.userData.GetUsers(ctx, name, email, pageNum, pageSize)
}

func (u *UserUseCase) Register(ctx context.Context, user *User) error {
	if user.Email == "" || user.Password == "" {
		return errors.New("email and password are required")
	}
	user.Status = UserStatus_USER_ENABLE
	return u.userData.Save(ctx, user)
}

func (u *UserUseCase) Disable(ctx context.Context, id int64) error {
	user, err := u.userData.GetUser(ctx, id)
	if err != nil {
		return err
	}
	user.Status = UserStatus_USER_DISABLE
	return u.userData.Save(ctx, user)
}

func (u *UserUseCase) SignIn(ctx context.Context, email, passwd string) (*User, *time.Time, error) {
	if email == u.conf.Auth.AdminEmail {
		if passwd != utils.Md5(u.conf.Auth.AdminPassword) {
			return nil, nil, errors.New(ErrUserPasswordError)
		}
		user := &User{
			Name:   email,
			Email:  email,
			Status: UserStatus_USER_ENABLE,
		}
		token, expires, err := GenerateJWT(user, u.conf.Auth.Exp, u.conf.Auth.Key)
		if err != nil {
			return nil, nil, err
		}
		user.AccessToken = token
		return user, expires, nil
	}
	user, err := u.userData.GetUserInfoByEmail(ctx, email)
	if err != nil {
		return nil, nil, err
	}
	if user.Id == 0 {
		return nil, nil, errors.New(ErrUserNotFound)
	}
	if user.Password != passwd {
		return nil, nil, errors.New(ErrUserPasswordError)
	}
	token, expires, err := GenerateJWT(user, u.conf.Auth.Exp, u.conf.Auth.Key)
	if err != nil {
		return nil, nil, err
	}
	user.AccessToken = token
	return user, expires, nil
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

func GenerateJWT(user *User, exp int32, authKey string) (string, *time.Time, error) {
	claims := jwtv5.MapClaims{
		"id":             user.Id,
		"name":           user.Name,
		"namespace":      user.Namespace,
		"email":          user.Email,
		"gitrepo_name":   user.GitrepoName,
		"imagerepo_name": user.ImagerepoName,
		"status":         user.Status,
		"workspace_id":   user.WorkspaceId,
	}

	expires := time.Now().Add(time.Hour * time.Duration(exp))
	claims["exp"] = expires.Unix()

	token, err := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, claims).SignedString([]byte(authKey))
	if err != nil {
		return "", &time.Time{}, err
	}

	return token, &expires, nil
}

func ValidateJWT(token, authKey string) (*User, error) {
	claims := jwtv5.MapClaims{}
	_, err := jwtv5.ParseWithClaims(token, &claims, func(token *jwtv5.Token) (interface{}, error) {
		return []byte(authKey), nil
	})
	if err != nil {
		return nil, err
	}
	exp, ok := claims["exp"]
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
