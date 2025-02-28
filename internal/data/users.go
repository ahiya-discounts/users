package data

import (
	"context"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
	"users/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
)

type Users struct {
	gorm.Model
	ID       uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primary_key"`
	Username string    `gorm:"not null;uniqueIndex"`
	Email    string    `gorm:"not null;uniqueIndex"`
	Phone    *string   `gorm:"not null;uniqueIndex"`
	Avatar   *string
}

type usersRepo struct {
	data *Data
	log  *log.Helper
}

func NewUsersRepo(data *Data, logger log.Logger) biz.UsersRepo {
	return &usersRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

func (r *usersRepo) Save(ctx context.Context, u *biz.Users) (*biz.Users, error) {
	_, span := otel.Tracer("users").Start(ctx, "Data Save")
	defer span.End()
	user := &Users{
		Username: *u.Username,
		Email:    *u.Email,
		Phone:    u.Phone,
		Avatar:   u.Avatar,
	}
	t := r.data.client.Save(user)

	if t.Error != nil {
		return nil, t.Error
	}
	resp := &biz.Users{
		ID:        user.ID.String(),
		Username:  &user.Username,
		Email:     &user.Email,
		Phone:     user.Phone,
		Avatar:    user.Avatar,
		CreatedAt: &user.CreatedAt,
		UpdatedAt: &user.UpdatedAt,
		DeletedAt: &user.DeletedAt.Time,
	}
	return resp, nil
}

func (r *usersRepo) Update(ctx context.Context, u *biz.Users) (*biz.Users, error) {
	_, span := otel.Tracer("users").Start(ctx, "Data Update")
	defer span.End()

	uid, err := uuid.Parse(u.ID)

	if err != nil {
		r.log.Warn(err.Error(), zap.Error(err))
		span.AddEvent(err.Error())
		return nil, err
	}
	user := &Users{
		Username: *u.Username,
		Email:    *u.Email,
		Phone:    u.Phone,
		Avatar:   u.Avatar,
	}
	t := r.data.client.Model(&Users{}).Where("id = ?", uid).Updates(user)
	if t.Error != nil {
		return nil, t.Error
	}

	resp := &biz.Users{
		ID:        user.ID.String(),
		Username:  &user.Username,
		Email:     &user.Email,
		Phone:     user.Phone,
		Avatar:    user.Avatar,
		CreatedAt: &user.CreatedAt,
		UpdatedAt: &user.UpdatedAt,
		DeletedAt: &user.DeletedAt.Time,
	}
	return resp, nil
}

func (r *usersRepo) FindByID(ctx context.Context, id uuid.UUID) (*biz.Users, error) {
	_, span := otel.Tracer("users").Start(ctx, "Data FindByID")
	defer span.End()
	user := &Users{
		ID: id,
	}
	t := r.data.client.First(&user)
	if t.Error != nil {
		return nil, t.Error
	}
	resp := &biz.Users{
		ID:       user.ID.String(),
		Username: &user.Username,
		Email:    &user.Email,
		Phone:    user.Phone,
		Avatar:   user.Avatar,
	}
	return resp, nil
}

func (r *usersRepo) ListAll(ctx context.Context, pp biz.PaginationParams, sp biz.SortParams) ([]biz.Users, error) {
	_, span := otel.Tracer("users").Start(ctx, "Data ListAll")
	defer span.End()
	offset := pp.PageSize * pp.Page

	var usersList []Users
	q := r.data.client.Offset(offset).Limit(pp.PageSize)
	if sp.SortBy != "" {
		sortOrder := "asc"
		if sp.SortOrder != "asc" && sp.SortOrder != "desc" {
			sortOrder = "asc"
		} else {
			sortOrder = sp.SortOrder
		}
		q = q.Order(sp.SortBy + " " + sortOrder)
	}
	q = q.Find(&usersList)

	if q.Error != nil {
		return nil, q.Error
	}

	var result []biz.Users
	for _, user := range usersList {
		result = append(result, biz.Users{
			ID:        user.ID.String(),
			Username:  &user.Username,
			Email:     &user.Email,
			Phone:     user.Phone,
			Avatar:    user.Avatar,
			CreatedAt: &user.CreatedAt,
			UpdatedAt: &user.UpdatedAt,
			DeletedAt: &user.DeletedAt.Time,
		})
	}

	res := make([]biz.Users, len(result))
	copy(res, result)

	return result, nil
}

func (r *usersRepo) Delete(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
	_, span := otel.Tracer("users").Start(ctx, "Data Delete")
	defer span.End()
	err := r.data.client.Delete(&Users{}, id).Error
	if err != nil {
		return id, err
	}
	return id, nil
}

func (r *usersRepo) Count(ctx context.Context) (int, error) {
	_, span := otel.Tracer("users").Start(ctx, "Data Count")
	defer span.End()
	var count int64

	t := r.data.client.Model(&Users{}).Count(&count)
	if t.Error != nil {
		return 0, t.Error
	}
	return int(count), nil
}
