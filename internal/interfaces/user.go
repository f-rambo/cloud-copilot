package interfaces

import (
	"context"

	"github.com/f-rambo/ocean/api/user/v1alpha1"
	"github.com/f-rambo/ocean/internal/biz"
	"github.com/f-rambo/ocean/internal/conf"
	"google.golang.org/protobuf/types/known/emptypb"
)

type UserInterface struct {
	v1alpha1.UnimplementedUserInterfaceServer
	uc       *biz.UserUseCase
	authConf *conf.Auth
}

func NewUserInterface(uc *biz.UserUseCase, conf *conf.Auth) *UserInterface {
	return &UserInterface{uc: uc, authConf: conf}
}

func (u *UserInterface) SignIn(ctx context.Context, in *v1alpha1.SignIn) (*v1alpha1.User, error) {
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
		Token:    token,
		ExpHour:  u.authConf.Exp,
	}, nil
}

func (u *UserInterface) SignOut(ctx context.Context, in *emptypb.Empty) (*v1alpha1.Msg, error) {
	return nil, nil
}

func (u *UserInterface) GetUserInfo(ctx context.Context, in *emptypb.Empty) (*v1alpha1.User, error) {
	user, err := u.uc.GetUserInfo(ctx)
	if err != nil {
		return nil, err
	}
	return &v1alpha1.User{
		Id:       user.ID,
		Email:    user.Email,
		Username: user.Name,
		Token:    user.PassWord,
		ExpHour:  u.authConf.Exp,
	}, nil
}
