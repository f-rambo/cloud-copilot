package data

import (
	"context"
	"fmt"

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

	if user.Id > 0 {
		var workspaceRoles []biz.WorkspaceRole
		if err = u.data.db.Where("user_id = ?", user.Id).Find(&workspaceRoles).Error; err != nil {
			u.log.Errorf("failed to get workspace roles for user %d: %v", user.Id, err)
		} else {
			user.WorkspaceRoles = workspaceRoles
		}
	}

	return user, nil
}

func (u *UserRepo) Save(ctx context.Context, user *biz.User) error {
	tx := u.data.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Save(user).Error; err != nil {
		tx.Rollback()
		return err
	}

	if len(user.WorkspaceRoles) > 0 && user.Id > 0 {
		var existingRoles []biz.WorkspaceRole
		if err := tx.Where("user_id = ?", user.Id).Find(&existingRoles).Error; err != nil {
			tx.Rollback()
			return err
		}

		existingMap := make(map[string]biz.WorkspaceRole)
		for _, role := range existingRoles {
			key := fmt.Sprintf("%d:%d", role.WorkspaceId, role.RoleId)
			existingMap[key] = role
		}

		toSave := make([]biz.WorkspaceRole, 0)
		newRoleKeys := make(map[string]bool)

		for _, role := range user.WorkspaceRoles {
			role.UserId = user.Id
			key := fmt.Sprintf("%d:%d", role.WorkspaceId, role.RoleId)
			newRoleKeys[key] = true

			if _, exists := existingMap[key]; !exists {
				toSave = append(toSave, role)
			}
		}

		toDelete := make([]int64, 0)
		for _, role := range existingRoles {
			key := fmt.Sprintf("%d:%d", role.WorkspaceId, role.RoleId)
			if _, exists := newRoleKeys[key]; !exists {
				toDelete = append(toDelete, role.Id)
			}
		}

		if len(toSave) > 0 {
			if err := tx.Create(&toSave).Error; err != nil {
				tx.Rollback()
				return err
			}
		}

		if len(toDelete) > 0 {
			if err := tx.Where("id IN (?)", toDelete).Delete(&biz.WorkspaceRole{}).Error; err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	return tx.Commit().Error
}

func (u *UserRepo) GetUser(ctx context.Context, id int64) (*biz.User, error) {
	user := &biz.User{}
	err := u.data.db.Where("id = ?", id).First(user).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	if user.Id > 0 {
		var workspaceRoles []biz.WorkspaceRole
		if err = u.data.db.Where("user_id = ?", user.Id).Find(&workspaceRoles).Error; err != nil {
			u.log.Errorf("failed to get workspace roles for user %d: %v", user.Id, err)
		} else {
			user.WorkspaceRoles = workspaceRoles
		}
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

	if len(users) > 0 {
		userIDs := make([]int64, 0, len(users))
		userMap := make(map[int64]*biz.User)
		for _, user := range users {
			userIDs = append(userIDs, user.Id)
			userMap[user.Id] = user
		}
		var workspaceRoles []biz.WorkspaceRole
		if err = u.data.db.Where("user_id IN (?)", userIDs).Find(&workspaceRoles).Error; err != nil {
			u.log.Errorf("failed to get workspace roles for users: %v", err)
		} else {
			for i := range workspaceRoles {
				if user, ok := userMap[workspaceRoles[i].UserId]; ok {
					user.WorkspaceRoles = append(user.WorkspaceRoles, workspaceRoles[i])
				}
			}
		}
	}

	return users, total, nil
}

func (u *UserRepo) DeleteUser(ctx context.Context, id int64) error {
	tx := u.data.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Where("user_id = ?", id).Delete(&biz.WorkspaceRole{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Delete(&biz.User{}, "id = ?", id).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (u *UserRepo) SaveRole(ctx context.Context, role *biz.Role) error {
	if err := u.data.db.Save(role).Error; err != nil {
		return err
	}
	if len(role.Permissions) > 0 && role.Id > 0 {
		var existingPermissions []biz.Permission
		if err := u.data.db.Where("role_id = ?", role.Id).Find(&existingPermissions).Error; err != nil {
			return err
		}

		existingMap := make(map[string]biz.Permission)
		for _, p := range existingPermissions {
			key := fmt.Sprintf("%d-%d-%d", p.RoleResourceType, p.ResourceId, p.ActionType)
			existingMap[key] = p
		}

		newMap := make(map[string]biz.Permission)
		for i := range role.Permissions {
			role.Permissions[i].RoleId = role.Id
			key := fmt.Sprintf("%d-%d-%d", role.Permissions[i].RoleResourceType,
				role.Permissions[i].ResourceId, role.Permissions[i].ActionType)
			newMap[key] = role.Permissions[i]
		}

		var deleteIDs []int64
		for key, p := range existingMap {
			if _, exists := newMap[key]; !exists {
				deleteIDs = append(deleteIDs, p.Id)
			}
		}

		if len(deleteIDs) > 0 {
			if err := u.data.db.Where("id IN (?)", deleteIDs).Delete(&biz.Permission{}).Error; err != nil {
				return err
			}
		}

		for key, newPerm := range newMap {
			if existingPerm, exists := existingMap[key]; exists {
				newPerm.Id = existingPerm.Id
			}
			if err := u.data.db.Save(&newPerm).Error; err != nil {
				return err
			}
		}
	}

	return nil
}

func (u *UserRepo) GetRoles(ctx context.Context, name string, page, size int) (roles []*biz.Role, total int64, err error) {
	roles = make([]*biz.Role, 0)
	db := u.data.db.Model(&biz.Role{})

	if name != "" {
		db = db.Where("name LIKE ?", "%"+name+"%")
	}

	err = db.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	if total == 0 {
		return roles, 0, nil
	}

	err = db.Offset((page - 1) * size).Limit(size).Find(&roles).Error
	if err != nil {
		return nil, 0, err
	}

	if len(roles) > 0 {
		roleIDs := make([]int64, 0, len(roles))
		roleMap := make(map[int64]*biz.Role)

		for _, role := range roles {
			roleIDs = append(roleIDs, role.Id)
			roleMap[role.Id] = role
		}

		var permissions []biz.Permission
		if err = u.data.db.Where("role_id IN (?)", roleIDs).Find(&permissions).Error; err == nil {
			for i := range permissions {
				if role, ok := roleMap[permissions[i].RoleId]; ok {
					role.Permissions = append(role.Permissions, permissions[i])
				}
			}
		}
	}

	return roles, total, nil
}

func (u *UserRepo) GetRole(ctx context.Context, id int64) (*biz.Role, error) {
	role := &biz.Role{}

	err := u.data.db.Where("id = ?", id).First(role).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}

	if role.Id > 0 {
		var permissions []biz.Permission
		if err = u.data.db.Where("role_id = ?", role.Id).Find(&permissions).Error; err != nil {
			return role, nil
		}
		role.Permissions = permissions
	}

	return role, nil
}

func (u *UserRepo) DeleteRole(ctx context.Context, id int64) error {
	tx := u.data.db.Begin()

	if err := tx.Where("role_id = ?", id).Delete(&biz.Permission{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Delete(&biz.Role{}, "id = ?", id).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}
