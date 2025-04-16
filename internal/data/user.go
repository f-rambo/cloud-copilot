package data

import (
	"context"

	"github.com/f-rambo/cloud-copilot/internal/biz"
	"github.com/f-rambo/cloud-copilot/internal/conf"
	"github.com/go-kratos/kratos/v2/log"
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

func (u *UserRepo) GetUser(ctx context.Context, id int64) (*biz.User, error) {
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
