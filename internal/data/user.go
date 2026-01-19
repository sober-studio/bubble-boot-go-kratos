package data

import (
	"context"
	"errors"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/sober-studio/bubble-boot-go-kratos/internal/biz"
	"github.com/sober-studio/bubble-boot-go-kratos/internal/data/model"
	"gorm.io/gorm"
)

var _ biz.UserRepo = (*userRepo)(nil)

type userRepo struct {
	data *Data
	log  *log.Helper
}

func NewUserRepo(data *Data, logger log.Logger) biz.UserRepo {
	return &userRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

func (r *userRepo) CreateUser(ctx context.Context, u *biz.User) (*biz.User, error) {
	user := &model.User{
		Username:     u.Username,
		PasswordHash: u.PasswordHash,
		IsAvailable:  &u.IsAvailable,
	}
	if u.Phone != "" {
		user.Phone = &u.Phone
	}
	if u.Nickname != "" {
		user.Nickname = &u.Nickname
	}

	if err := r.data.Q(ctx).User.WithContext(ctx).Create(user); err != nil {
		return nil, err
	}

	return r.toBiz(user), nil
}

func (r *userRepo) GetUserByUsername(ctx context.Context, username string) (*biz.User, error) {
	u := r.data.Q(ctx).User
	user, err := u.WithContext(ctx).Where(u.Username.Eq(username)).First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, biz.ErrUserNotFound
		}
		return nil, err
	}
	return r.toBiz(user), nil
}

func (r *userRepo) GetUserByPhone(ctx context.Context, phone string) (*biz.User, error) {
	u := r.data.Q(ctx).User
	user, err := u.WithContext(ctx).Where(u.Phone.Eq(phone)).First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, biz.ErrUserNotFound
		}
		return nil, err
	}
	return r.toBiz(user), nil
}

func (r *userRepo) GetUserByID(ctx context.Context, id int64) (*biz.User, error) {
	var user model.User
	if err := r.data.db.WithContext(ctx).Where("id = ?", id).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, biz.ErrUserNotFound
		}
		return nil, err
	}
	return r.toBiz(&user), nil
}

func (r *userRepo) UpdatePassword(ctx context.Context, id int64, passwordHash string) error {
	return r.data.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ?", id).
		Update("password_hash", passwordHash).Error
}

func (r *userRepo) UpdatePhone(ctx context.Context, id int64, phone string) error {
	return r.data.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ?", id).
		Update("phone", phone).Error
}

func (r *userRepo) toBiz(u *model.User) *biz.User {
	phone := ""
	if u.Phone != nil {
		phone = *u.Phone
	}
	nickname := ""
	if u.Nickname != nil {
		nickname = *u.Nickname
	}
	isAvailable := false
	if u.IsAvailable != nil {
		isAvailable = *u.IsAvailable
	}

	return &biz.User{
		ID:           u.ID,
		Username:     u.Username,
		PasswordHash: u.PasswordHash,
		Phone:        phone,
		Nickname:     nickname,
		IsAvailable:  isAvailable,
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
	}
}
