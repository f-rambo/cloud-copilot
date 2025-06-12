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

func UserstatusFromString(statusStr string) UserStatus {
	switch statusStr {
	case "USER_INIT":
		return UserStatus_USER_INIT
	case "USER_ENABLE":
		return UserStatus_USER_ENABLE
	case "USER_DISABLE":
		return UserStatus_USER_DISABLE
	case "USER_DELETED":
		return UserStatus_USER_DELETED
	default:
		return UserStatus_USER_INIT
	}
}

type RoleType int32

const (
	RoleType_SYSTEM_ADMIN    RoleType = 1
	RoleType_WORKSPACE_ADMIN RoleType = 2
	RoleType_PROJECT_ADMIN   RoleType = 3
	RoleType_DEVELOPER       RoleType = 4
	RoleType_VIEWER          RoleType = 5
	RoleType_CUSTOM          RoleType = 6
)

func (rt RoleType) String() string {
	switch rt {
	case RoleType_SYSTEM_ADMIN:
		return "SYSTEM_ADMIN"
	case RoleType_WORKSPACE_ADMIN:
		return "WORKSPACE_ADMIN"
	case RoleType_PROJECT_ADMIN:
		return "PROJECT_ADMIN"
	case RoleType_DEVELOPER:
		return "DEVELOPER"
	case RoleType_VIEWER:
		return "VIEWER"
	case RoleType_CUSTOM:
		return "CUSTOM"
	default:
		return "UNKNOWN"
	}
}

func RoleTypeFromString(roleTypeStr string) RoleType {
	switch roleTypeStr {
	case "SYSTEM_ADMIN":
		return RoleType_SYSTEM_ADMIN
	case "WORKSPACE_ADMIN":
		return RoleType_WORKSPACE_ADMIN
	case "PROJECT_ADMIN":
		return RoleType_PROJECT_ADMIN
	case "DEVELOPER":
		return RoleType_DEVELOPER
	case "VIEWER":
		return RoleType_VIEWER
	case "CUSTOM":
		return RoleType_CUSTOM
	default:
		return RoleType_SYSTEM_ADMIN
	}
}

type RoleResourceType int32

const (
	RoleResourceType_CLUSTER   RoleResourceType = 1 // 集群资源
	RoleResourceType_WORKSPACE RoleResourceType = 2 // 工作空间资源
	RoleResourceType_PROJECT   RoleResourceType = 3 // 项目资源
	RoleResourceType_SERVICE   RoleResourceType = 4 // 服务资源
	RoleResourceType_SYSTEM    RoleResourceType = 5 // 系统资源
	RoleResourceType_APP       RoleResourceType = 6 // 应用资源
)

func (rt RoleResourceType) String() string {
	switch rt {
	case RoleResourceType_CLUSTER:
		return "CLUSTER"
	case RoleResourceType_WORKSPACE:
		return "WORKSPACE"
	case RoleResourceType_PROJECT:
		return "PROJECT"
	case RoleResourceType_SERVICE:
		return "SERVICE"
	case RoleResourceType_SYSTEM:
		return "SYSTEM"
	case RoleResourceType_APP:
		return "APP"
	default:
		return "UNKNOWN"
	}
}

func RoleResourceTypeFromString(resourceTypeStr string) RoleResourceType {
	switch resourceTypeStr {
	case "CLUSTER":
		return RoleResourceType_CLUSTER
	case "WORKSPACE":
		return RoleResourceType_WORKSPACE
	case "PROJECT":
		return RoleResourceType_PROJECT
	case "SERVICE":
		return RoleResourceType_SERVICE
	case "SYSTEM":
		return RoleResourceType_SYSTEM
	case "APP":
		return RoleResourceType_APP
	default:
		return RoleResourceType_CLUSTER
	}
}

type ActionType int32

const (
	ActionType_VIEW    ActionType = 1 // 查看
	ActionType_CREATE  ActionType = 2 // 创建
	ActionType_UPDATE  ActionType = 3 // 更新
	ActionType_DELETE  ActionType = 4 // 删除
	ActionType_EXECUTE ActionType = 5 // 执行
	ActionType_MANAGE  ActionType = 6 // 管理(包含所有权限)
)

func (at ActionType) String() string {
	switch at {
	case ActionType_VIEW:
		return "VIEW"
	case ActionType_CREATE:
		return "CREATE"
	case ActionType_UPDATE:
		return "UPDATE"
	case ActionType_DELETE:
		return "DELETE"
	case ActionType_EXECUTE:
		return "EXECUTE"
	case ActionType_MANAGE:
		return "MANAGE"
	default:
		return "UNKNOWN"
	}
}

func ActionTypeFromString(actionTypeStr string) ActionType {
	switch actionTypeStr {
	case "VIEW":
		return ActionType_VIEW
	case "CREATE":
		return ActionType_CREATE
	case "UPDATE":
		return ActionType_UPDATE
	case "DELETE":
		return ActionType_DELETE
	case "EXECUTE":
		return ActionType_EXECUTE
	case "MANAGE":
		return ActionType_MANAGE
	default:
		return ActionType_VIEW
	}
}

type User struct {
	Id             int64           `json:"id,omitempty" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Name           string          `json:"name,omitempty" gorm:"column:name;default:'';NOT NULL"`
	Email          string          `json:"email,omitempty" gorm:"column:email;default:'';NOT NULL"`
	Password       string          `json:"password,omitempty" gorm:"column:password;default:'';NOT NULL"`
	Status         UserStatus      `json:"status,omitempty" gorm:"column:status;default:0;NOT NULL"`
	AccessToken    string          `json:"access_token,omitempty" gorm:"-"`
	Avatar         []byte          `json:"avatar,omitempty" gorm:"type:bytea;column:avatar"`
	Phone          string          `json:"phone,omitempty" gorm:"column:phone;default:'';NOT NULL"`
	Department     string          `json:"department,omitempty" gorm:"column:department;default:'';NOT NULL"`
	WorkspaceRoles []WorkspaceRole `json:"workspace_roles,omitempty" gorm:"-"`
}

type Role struct {
	Id          int64        `json:"id,omitempty" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Name        string       `json:"name,omitempty" gorm:"column:name;default:'';NOT NULL"`
	Verbs       string       `json:"verbs,omitempty" gorm:"column:verbs;default:'';NOT NULL"`
	Resources   string       `json:"resources,omitempty" gorm:"column:resources;default:'';NOT NULL"`
	Description string       `json:"description,omitempty" gorm:"column:description;default:'';NOT NULL"`
	WorkspaceId int64        `json:"workspace_id,omitempty" gorm:"column:workspace_id;default:0;NOT NULL"`
	RoleType    RoleType     `json:"role_type,omitempty" gorm:"column:role_type;default:0;NOT NULL"`
	Permissions []Permission `json:"permissions,omitempty" gorm:"-"`
}

// 权限表
type Permission struct {
	Id               int64            `json:"id,omitempty" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	RoleResourceType RoleResourceType `json:"role_resource_type,omitempty" gorm:"column:role_resource_type;default:0;NOT NULL"`
	ResourceId       int64            `json:"resource_id,omitempty" gorm:"column:resource_id;default:0;NOT NULL"` // 资源ID，0表示所有资源
	ActionType       ActionType       `json:"action_type,omitempty" gorm:"column:action_type;default:0;NOT NULL"`
	RoleId           int64            `json:"role_id,omitempty" gorm:"column:role_id;default:0;NOT NULL"`
}

type WorkspaceRole struct {
	Id          int64 `json:"id,omitempty" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	WorkspaceId int64 `json:"workspace_id,omitempty" gorm:"column:workspace_id;default:0;NOT NULL"`
	UserId      int64 `json:"user_id,omitempty" gorm:"column:user_id;default:0;NOT NULL"`
	RoleId      int64 `json:"role_id,omitempty" gorm:"column:role_id;default:0;NOT NULL"`
}

type UserData interface {
	GetUserInfoByEmail(ctx context.Context, email string) (*User, error)
	GetUser(ctx context.Context, id int64) (*User, error)
	Save(ctx context.Context, user *User) error
	GetUsers(ctx context.Context, username, email string, pageNum, pageSize int) (users []*User, total int64, err error)
	DeleteUser(ctx context.Context, id int64) error
	SaveRole(ctx context.Context, role *Role) error
	GetRoles(ctx context.Context, name string, page, size int) (roles []*Role, total int64, err error)
	GetRole(ctx context.Context, id int64) (*Role, error)
	DeleteRole(ctx context.Context, id int64) error
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

func (u *UserUseCase) Enable(ctx context.Context, user *User) error {
	user.Status = UserStatus_USER_ENABLE
	return u.Save(ctx, user)
}

func (u *UserUseCase) Disable(ctx context.Context, user *User) error {
	user.Status = UserStatus_USER_DISABLE
	return u.Save(ctx, user)
}

func (u *UserUseCase) Save(ctx context.Context, user *User) error {
	if user.Email == "" || user.Password == "" {
		return errors.New("email and password are required")
	}
	if user.Id == 0 {
		userRes, err := u.userData.GetUserInfoByEmail(ctx, user.Email)
		if err != nil {
			return err
		}
		if userRes != nil && userRes.Id != 0 {
			return errors.New("user already exists")
		}
	}
	return u.userData.Save(ctx, user)
}

func (u *UserUseCase) SignIn(ctx context.Context, email, passwd string) (*User, *time.Time, error) {
	if email == u.conf.Auth.AdminEmail {
		if passwd != utils.Md5(u.conf.Auth.AdminPassword) {
			return nil, nil, errors.New(ErrUserPasswordError)
		}
		user := &User{
			Name:   "Admin",
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

func (u *UserUseCase) DeleteUser(ctx context.Context, id int64) error {
	return u.userData.DeleteUser(ctx, id)
}

func (u *UserUseCase) GetRole(ctx context.Context, id int64) (*Role, error) {
	return u.userData.GetRole(ctx, id)
}

func (u *UserUseCase) SaveRole(ctx context.Context, role *Role) error {
	return u.userData.SaveRole(ctx, role)
}

func (u *UserUseCase) GetRoles(ctx context.Context, name string, page, size int) (roles []*Role, total int64, err error) {
	return u.userData.GetRoles(ctx, name, page, size)
}

func (u *UserUseCase) DeleteRole(ctx context.Context, id int64) error {
	return u.userData.DeleteRole(ctx, id)
}

func GenerateJWT(user *User, exp int32, authKey string) (string, *time.Time, error) {
	claims := jwtv5.MapClaims{
		"id":     user.Id,
		"name":   user.Name,
		"email":  user.Email,
		"phone":  user.Phone,
		"status": user.Status,
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
