package interfaces

import (
	"context"
	"strings"

	"github.com/f-rambo/cloud-copilot/api/common"
	"github.com/f-rambo/cloud-copilot/api/user/v1alpha1"
	"github.com/f-rambo/cloud-copilot/internal/biz"
	"github.com/f-rambo/cloud-copilot/internal/conf"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/emptypb"
)

type UserInterface struct {
	v1alpha1.UnimplementedUserInterfaceServer
	uc *biz.UserUseCase
}

func NewUserInterface(uc *biz.UserUseCase, conf *conf.Bootstrap) *UserInterface {
	return &UserInterface{uc: uc}
}

func (u *UserInterface) SignIn(ctx context.Context, request *v1alpha1.SignIn) (*v1alpha1.User, error) {
	user := &biz.User{
		Name:        request.Username,
		Email:       request.Email,
		Password:    request.Password,
		AccessToken: request.AccessToken,
		SignType:    biz.UserSignType(request.SignType),
	}
	err := u.uc.SignIn(ctx, user)
	if err != nil {
		return nil, err
	}
	return &v1alpha1.User{
		Id:             user.Id,
		Email:          user.Email,
		Username:       user.Name,
		AccessToken:    user.AccessToken,
		Status:         int32(user.Status),
		StatusString:   user.Status.String(),
		UpdatedAt:      user.UpdatedAt.AsTime().Format("2006-01-02 15:04:05"),
		SignType:       int32(user.SignType),
		SignTypeString: user.SignType.String(),
	}, nil
}

func (u *UserInterface) GetUserInfo(ctx context.Context, _ *emptypb.Empty) (*v1alpha1.User, error) {
	user, err := u.uc.GetUserInfo(ctx)
	if err != nil {
		return nil, err
	}
	return &v1alpha1.User{
		Id:          user.Id,
		Email:       user.Email,
		Username:    user.Name,
		AccessToken: user.AccessToken,
		Status:      int32(user.Status),
		UpdatedAt:   user.UpdatedAt.AsTime().Format("2006-01-02 15:04:05"),
	}, nil
}

func (u *UserInterface) GetUsers(ctx context.Context, request *v1alpha1.UsersRequest) (*v1alpha1.Users, error) {
	users, total, err := u.uc.GetUsers(ctx, request.Username, request.Email, int(request.PageNumber), int(request.PageSize))
	if err != nil {
		return nil, err
	}
	var userList []*v1alpha1.User
	for _, user := range users {
		userList = append(userList, &v1alpha1.User{
			Id:             user.Id,
			Email:          user.Email,
			Username:       user.Name,
			AccessToken:    user.AccessToken,
			Status:         int32(user.Status),
			SignType:       int32(user.SignType),
			StatusString:   strings.ToUpper(user.Status.String()),
			SignTypeString: strings.ToUpper(user.SignType.String()),
			UpdatedAt:      user.UpdatedAt.AsTime().Format("2006-01-02 15:04:05"),
		})
	}
	return &v1alpha1.Users{
		Users:      userList,
		TotalCount: int32(total),
	}, nil
}

func (u *UserInterface) SaveUser(ctx context.Context, request *v1alpha1.User) (*v1alpha1.User, error) {
	user := &biz.User{
		Id:       request.Id,
		Email:    request.Email,
		Name:     request.Username,
		Status:   biz.UserStatus(request.Status),
		SignType: biz.UserSignType(request.SignType),
	}
	err := u.uc.Save(ctx, user)
	if err != nil {
		return nil, err
	}
	return &v1alpha1.User{
		Id:           user.Id,
		Email:        user.Email,
		Username:     user.Name,
		AccessToken:  user.AccessToken,
		Status:       int32(user.Status),
		StatusString: user.Status.String(),
		UpdatedAt:    user.UpdatedAt.AsTime().Format("2006-01-02 15:04:05"),
	}, nil
}

func (u *UserInterface) DeleteUser(ctx context.Context, request *v1alpha1.User) (*common.Msg, error) {
	if request.Id == 0 {
		return nil, errors.New("id is required")
	}
	err := u.uc.DeleteUser(ctx, request.Id)
	if err != nil {
		return nil, err
	}
	return common.Response(), nil
}
