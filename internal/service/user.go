package service

import (
	"context"

	"github.com/f-rambo/ocean/api/user/v1alpha1"
	"github.com/f-rambo/ocean/internal/biz"
	"google.golang.org/protobuf/types/known/emptypb"
)

type UserService struct {
	v1alpha1.UnimplementedUserServiceServer
	uc *biz.UserUseCase
}

func NewUserService(uc *biz.UserUseCase) *UserService {
	return &UserService{uc: uc}
}

func (u *UserService) SignIn(ctx context.Context, in *v1alpha1.SignIn) (*v1alpha1.User, error) {
	token, err := u.uc.SignIn(ctx, in.Email, in.Password)
	if err != nil {
		return nil, err
	}
	user, err := u.uc.CheckJWT(ctx, token)
	if err != nil {
		return nil, err
	}
	return &v1alpha1.User{
		Id:       user.ID,
		Email:    user.Email,
		Username: user.Name,
	}, nil
}

func (u *UserService) SignOut(ctx context.Context, in *emptypb.Empty) (*v1alpha1.Msg, error) {
	err := u.uc.SignOut(ctx)
	if err != nil {
		return nil, err
	}
	return &v1alpha1.Msg{Reason: v1alpha1.ErrorReason_SUCCEED, Message: "ok"}, nil
}

func (u *UserService) GetUserInfo(ctx context.Context, in *emptypb.Empty) (*v1alpha1.User, error) {
	user, err := u.uc.GetUserInfo(ctx)
	if err != nil {
		return nil, err
	}
	return &v1alpha1.User{
		Id:       user.ID,
		Email:    user.Email,
		Username: user.Name,
	}, nil
}
