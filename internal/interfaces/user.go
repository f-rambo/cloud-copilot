package interfaces

import (
	"context"
	"strconv"
	"strings"

	"github.com/f-rambo/cloud-copilot/api/common"
	"github.com/f-rambo/cloud-copilot/api/user/v1alpha1"
	"github.com/f-rambo/cloud-copilot/internal/biz"
	"github.com/f-rambo/cloud-copilot/internal/conf"
	"github.com/pkg/errors"
)

type UserInterface struct {
	v1alpha1.UnimplementedUserInterfaceServer
	uc *biz.UserUseCase
}

func NewUserInterface(uc *biz.UserUseCase, conf *conf.Bootstrap) *UserInterface {
	return &UserInterface{uc: uc}
}

func (u *UserInterface) SignIn(ctx context.Context, request *v1alpha1.SignInRequest) (*v1alpha1.User, error) {
	user, expires, err := u.uc.SignIn(ctx, request.Email, request.Password)
	if err != nil {
		return nil, err
	}
	data := u.userBizToInterface(user)
	data.Expires = strconv.FormatInt(expires.Unix(), 10)
	return data, nil
}

func (u *UserInterface) GetUsers(ctx context.Context, request *v1alpha1.UsersRequest) (*v1alpha1.Users, error) {
	users, total, err := u.uc.GetUsers(ctx, request.Username, request.Email, int(request.PageNumber), int(request.PageSize))
	if err != nil {
		return nil, err
	}
	var userList []*v1alpha1.User
	for _, user := range users {
		userList = append(userList, u.userBizToInterface(user))
	}
	return &v1alpha1.Users{
		Users:      userList,
		TotalCount: int32(total),
	}, nil
}

func (u *UserInterface) SaveUser(ctx context.Context, request *v1alpha1.User) (*common.Msg, error) {
	user := u.userInterfaceToBiz(request)
	err := u.uc.Save(ctx, user)
	if err != nil {
		return nil, err
	}
	return common.Response(), nil
}

func (u *UserInterface) DeleteUser(ctx context.Context, request *v1alpha1.User) (*common.Msg, error) {
	if request.Id == 0 {
		return nil, errors.New("id is required")
	}
	user, err := u.uc.GetUser(ctx, int64(request.Id))
	if err != nil {
		return nil, err
	}
	if user == nil || user.Id == 0 {
		return common.Response(), nil
	}
	err = u.uc.DeleteUser(ctx, int64(request.Id))
	if err != nil {
		return nil, err
	}
	return common.Response(), nil
}

// EnableUser(UserIdRequest) returns (common.Msg)
func (u *UserInterface) EnableUser(ctx context.Context, request *v1alpha1.UserIdRequest) (*common.Msg, error) {
	if request.UserId == 0 {
		return nil, errors.New("id is required")
	}
	user, err := u.uc.GetUser(ctx, int64(request.UserId))
	if err != nil {
		return nil, err
	}
	if user == nil || user.Id == 0 {
		return nil, errors.New("user not found")
	}
	err = u.uc.Enable(ctx, user)
	if err != nil {
		return nil, err
	}
	return common.Response(), nil
}

// DisableUser(UserIdRequest) returns (common.Msg)
func (u *UserInterface) DisableUser(ctx context.Context, request *v1alpha1.UserIdRequest) (*common.Msg, error) {
	if request.UserId == 0 {
		return nil, errors.New("id is required")
	}
	user, err := u.uc.GetUser(ctx, int64(request.UserId))
	if err != nil {
		return nil, err
	}
	if user == nil || user.Id == 0 {
		return nil, errors.New("user not found")
	}
	err = u.uc.Disable(ctx, user)
	if err != nil {
		return nil, err
	}
	return common.Response(), nil
}

// SaveRole(Role) returns (common.Msg)
func (u *UserInterface) SaveRole(ctx context.Context, request *v1alpha1.Role) (*common.Msg, error) {
	if request.Name == "" {
		return nil, errors.New("name is required")
	}
	role := u.roleInterfaceToBiz(request)
	err := u.uc.SaveRole(ctx, role)
	if err != nil {
		return nil, err
	}
	return common.Response(), nil
}

// GetRoles(RolesRequest) returns (Roles)
func (u *UserInterface) GetRoles(ctx context.Context, request *v1alpha1.RolesRequest) (*v1alpha1.Roles, error) {
	if request.PageNumber == 0 {
		request.PageNumber = 1
	}
	if request.PageSize == 0 {
		request.PageSize = 10
	}
	request.Name = strings.TrimSpace(request.Name)
	roles, total, err := u.uc.GetRoles(ctx, request.Name, int(request.PageNumber), int(request.PageSize))
	if err != nil {
		return nil, err
	}
	var roleList []*v1alpha1.Role
	for _, role := range roles {
		roleList = append(roleList, u.roleBizToInterface(role))
	}
	return &v1alpha1.Roles{
		Roles:      roleList,
		TotalCount: int32(total),
	}, nil
}

// GetRole(RoleIdRequest) returns (Role)
func (u *UserInterface) GetRole(ctx context.Context, request *v1alpha1.RoleIdRequest) (*v1alpha1.Role, error) {
	if request.RoleId == 0 {
		return nil, errors.New("id is required")
	}
	role, err := u.uc.GetRole(ctx, int64(request.RoleId))
	if err != nil {
		return nil, err
	}
	if role == nil || role.Id == 0 {
		return nil, errors.New("role not found")
	}
	return u.roleBizToInterface(role), nil
}

// DeleteRole(RoleIdRequest) returns (common.Msg)
func (u *UserInterface) DeleteRole(ctx context.Context, request *v1alpha1.RoleIdRequest) (*common.Msg, error) {
	if request.RoleId == 0 {
		return nil, errors.New("id is required")
	}
	role, err := u.uc.GetRole(ctx, int64(request.RoleId))
	if err != nil {
		return nil, err
	}
	if role == nil || role.Id == 0 {
		return common.Response(), nil
	}
	err = u.uc.DeleteRole(ctx, int64(request.RoleId))
	if err != nil {
		return nil, err
	}
	return common.Response(), nil
}

func (u *UserInterface) userBizToInterface(user *biz.User) *v1alpha1.User {
	if user == nil {
		return nil
	}

	result := &v1alpha1.User{
		Id:         int32(user.Id),
		Name:       user.Name,
		Email:      user.Email,
		Password:   user.Password,
		Token:      user.AccessToken,
		Status:     user.Status.String(),
		Phone:      user.Phone,
		Department: user.Department,
		Avatar:     user.Avatar,
	}

	// 处理 WorkspaceRoles 字段
	if len(user.WorkspaceRoles) > 0 {
		result.WorkspaceRoles = make([]*v1alpha1.WorkspaceRole, 0, len(user.WorkspaceRoles))
		for _, role := range user.WorkspaceRoles {
			result.WorkspaceRoles = append(result.WorkspaceRoles, &v1alpha1.WorkspaceRole{
				Id:          int32(role.Id),
				WorkspaceId: int32(role.WorkspaceId),
				UserId:      int32(role.UserId),
				RoleId:      int32(role.RoleId),
			})
		}
	}

	return result
}

func (u *UserInterface) userInterfaceToBiz(user *v1alpha1.User) *biz.User {
	if user == nil {
		return nil
	}

	result := &biz.User{
		Id:          int64(user.Id),
		Name:        user.Name,
		Email:       user.Email,
		Password:    user.Password,
		AccessToken: user.Token,
		Phone:       user.Phone,
		Department:  user.Department,
		Avatar:      user.Avatar,
		Status:      biz.UserstatusFromString(user.Status),
	}

	// 处理 WorkspaceRoles 字段
	if len(user.WorkspaceRoles) > 0 {
		result.WorkspaceRoles = make([]biz.WorkspaceRole, 0, len(user.WorkspaceRoles))
		for _, role := range user.WorkspaceRoles {
			result.WorkspaceRoles = append(result.WorkspaceRoles, biz.WorkspaceRole{
				Id:          int64(role.Id),
				WorkspaceId: int64(role.WorkspaceId),
				UserId:      int64(role.UserId),
				RoleId:      int64(role.RoleId),
			})
		}
	}

	return result
}

func (u *UserInterface) roleBizToInterface(role *biz.Role) *v1alpha1.Role {
	if role == nil {
		return nil
	}

	result := &v1alpha1.Role{
		Id:          int32(role.Id),
		Name:        role.Name,
		Verbs:       role.Verbs,
		Resources:   role.Resources,
		Description: role.Description,
		WorkspaceId: int32(role.WorkspaceId),
		RoleType:    role.RoleType.String(),
	}

	if len(role.Permissions) > 0 {
		result.Permissions = make([]*v1alpha1.Permission, 0, len(role.Permissions))
		for _, perm := range role.Permissions {
			permission := &v1alpha1.Permission{
				Id:         int32(perm.Id),
				ResourceId: int32(perm.ResourceId),
				RoleId:     int32(perm.RoleId),
			}

			permission.RoleResourceType = perm.RoleResourceType.String()

			permission.ActionType = perm.ActionType.String()

			result.Permissions = append(result.Permissions, permission)
		}
	}

	return result
}

func (u *UserInterface) roleInterfaceToBiz(role *v1alpha1.Role) *biz.Role {
	if role == nil {
		return nil
	}

	result := &biz.Role{
		Id:          int64(role.Id),
		Name:        role.Name,
		Verbs:       role.Verbs,
		Resources:   role.Resources,
		Description: role.Description,
		WorkspaceId: int64(role.WorkspaceId),
		RoleType:    biz.RoleTypeFromString(role.RoleType),
	}

	// 处理 Permissions 字段
	if len(role.Permissions) > 0 {
		result.Permissions = make([]biz.Permission, 0, len(role.Permissions))
		for _, perm := range role.Permissions {
			permission := biz.Permission{
				Id:               int64(perm.Id),
				ResourceId:       int64(perm.ResourceId),
				RoleId:           int64(perm.RoleId),
				RoleResourceType: biz.RoleResourceTypeFromString(perm.RoleResourceType),
				ActionType:       biz.ActionTypeFromString(perm.ActionType),
			}
			result.Permissions = append(result.Permissions, permission)
		}
	}

	return result
}
