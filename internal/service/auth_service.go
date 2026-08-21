package service

import (
	"context"
	"strings"

	"go03-volunteer-activity/internal/auth"
	"go03-volunteer-activity/internal/model"
	"go03-volunteer-activity/internal/policy"
	"go03-volunteer-activity/internal/store"
	"go03-volunteer-activity/internal/validate"
)

type AuthService struct {
	store    store.Store
	hasher   *auth.PasswordHasher
	sessions *auth.SessionManager
	clock    Clock
	notify   *NotifyService
}

func NewAuthService(s store.Store, hasher *auth.PasswordHasher, sessions *auth.SessionManager, clock Clock, notify *NotifyService) *AuthService {
	return &AuthService{store: s, hasher: hasher, sessions: sessions, clock: clock, notify: notify}
}

func (s *AuthService) Register(ctx context.Context, req model.RegisterRequest) (model.LoginResponse, error) {
	username := strings.ToLower(validate.Trim(req.Username))
	if !validate.UsernameOK(username) {
		return model.LoginResponse{}, model.ErrInvalidUsername
	}
	if !validate.PasswordOK(req.Password) {
		return model.LoginResponse{}, model.ErrInvalidPassword
	}
	display := validate.SanitizePlain(req.DisplayName)
	if !validate.InRange(display, 1, policy.DisplayNameMax) {
		return model.LoginResponse{}, model.ErrInvalidDisplayName
	}
	if req.Phone != "" && !validate.InRange(req.Phone, 0, 20) {
		return model.LoginResponse{}, model.ErrInvalidPhone
	}
	if req.Age < 0 || req.Age > 120 {
		return model.LoginResponse{}, model.ErrInvalidAge
	}
	if _, err := s.store.GetUserByUsername(ctx, username); err == nil {
		return model.LoginResponse{}, model.ErrAlreadyExists
	}
	salt, hash, it, err := s.hasher.Hash(req.Password)
	if err != nil {
		return model.LoginResponse{}, err
	}
	now := s.clock.Now()
	u, err := s.store.CreateUser(ctx, model.User{
		Username:     username,
		DisplayName:  display,
		Role:         model.RoleVolunteer,
		Status:       model.UserActive,
		PasswordSalt: salt,
		PasswordHash: hash,
		Iterations:   it,
		Phone:        validate.Trim(req.Phone),
		Age:          req.Age,
		CreatedAt:    now,
		UpdatedAt:    now,
	})
	if err != nil {
		return model.LoginResponse{}, err
	}
	token, err := s.sessions.Create(u)
	if err != nil {
		return model.LoginResponse{}, err
	}
	_ = s.notify.Push(ctx, u.ID, model.NotifyActivity, "欢迎加入志愿者平台", "浏览广场报名活动，完成签到后可积累工时与证书。", u.ID)
	return model.LoginResponse{Token: token, User: u.Public()}, nil
}

func (s *AuthService) Login(ctx context.Context, req model.LoginRequest) (model.LoginResponse, error) {
	u, err := s.store.GetUserByUsername(ctx, strings.ToLower(validate.Trim(req.Username)))
	if err != nil {
		return model.LoginResponse{}, model.ErrInvalidCredentials
	}
	if u.IsBanned() {
		return model.LoginResponse{}, model.ErrAccountBanned
	}
	if !s.hasher.Verify(req.Password, u.PasswordSalt, u.PasswordHash, u.Iterations) {
		return model.LoginResponse{}, model.ErrInvalidCredentials
	}
	token, err := s.sessions.Create(u)
	if err != nil {
		return model.LoginResponse{}, err
	}
	return model.LoginResponse{Token: token, User: u.Public()}, nil
}

func (s *AuthService) Logout(_ context.Context, token string) {
	s.sessions.Invalidate(token)
}

func (s *AuthService) Me(ctx context.Context, u model.User) (model.MeResponse, error) {
	unread, _ := s.store.CountUnreadNotifications(ctx, u.ID)
	tier, remain := policy.NextCertificateTier(u.TotalMinutes)
	return model.MeResponse{
		User:              u.Public(),
		UnreadNotify:      unread,
		NextCertTier:      tier,
		MinutesToNextCert: remain,
	}, nil
}

type UserService struct {
	store    store.Store
	hasher   *auth.PasswordHasher
	sessions *auth.SessionManager
	clock    Clock
}

func NewUserService(s store.Store, hasher *auth.PasswordHasher, sessions *auth.SessionManager, clock Clock) *UserService {
	return &UserService{store: s, hasher: hasher, sessions: sessions, clock: clock}
}

func (s *UserService) UpdateProfile(ctx context.Context, actor model.User, req model.UpdateProfileRequest) (model.PublicUser, error) {
	if err := requireActiveWriter(actor); err != nil {
		return model.PublicUser{}, err
	}
	display := validate.SanitizePlain(req.DisplayName)
	if !validate.InRange(display, 1, policy.DisplayNameMax) {
		return model.PublicUser{}, model.ErrInvalidDisplayName
	}
	if req.Phone != "" && !validate.InRange(req.Phone, 0, 20) {
		return model.PublicUser{}, model.ErrInvalidPhone
	}
	if req.Bio != "" && !validate.InRange(req.Bio, 0, 200) {
		return model.PublicUser{}, model.ErrInvalidBio
	}
	if req.Age < 0 || req.Age > 120 {
		return model.PublicUser{}, model.ErrInvalidAge
	}
	actor.DisplayName = display
	actor.Phone = validate.Trim(req.Phone)
	actor.Bio = validate.SanitizePlain(req.Bio)
	actor.Age = req.Age
	actor.Skills = req.Skills
	actor.UpdatedAt = s.clock.Now()
	u, err := s.store.UpdateUser(ctx, actor)
	if err != nil {
		return model.PublicUser{}, err
	}
	return u.Public(), nil
}

func (s *UserService) ChangePassword(ctx context.Context, actor model.User, token string, req model.ChangePasswordRequest) error {
	if !s.hasher.Verify(req.OldPassword, actor.PasswordSalt, actor.PasswordHash, actor.Iterations) {
		return model.ErrInvalidCredentials
	}
	if !validate.PasswordOK(req.NewPassword) {
		return model.ErrInvalidPassword
	}
	salt, hash, it, err := s.hasher.Hash(req.NewPassword)
	if err != nil {
		return err
	}
	actor.PasswordSalt = salt
	actor.PasswordHash = hash
	actor.Iterations = it
	actor.UpdatedAt = s.clock.Now()
	if _, err := s.store.UpdateUser(ctx, actor); err != nil {
		return err
	}
	s.sessions.InvalidateByUser(actor.ID)
	return nil
}

func (s *UserService) List(ctx context.Context, actor model.User, f model.UserFilter) ([]model.PublicUser, error) {
	if !actor.IsAdmin() {
		return nil, model.ErrForbidden
	}
	users, err := s.store.ListUsers(ctx, f)
	if err != nil {
		return nil, err
	}
	out := make([]model.PublicUser, 0, len(users))
	for _, u := range users {
		out = append(out, u.Public())
	}
	return out, nil
}

func (s *UserService) Get(ctx context.Context, id string) (model.PublicUser, error) {
	u, err := s.store.GetUserByID(ctx, id)
	if err != nil {
		return model.PublicUser{}, err
	}
	return u.Public(), nil
}

func (s *UserService) Create(ctx context.Context, actor model.User, req model.CreateUserRequest) (model.PublicUser, error) {
	if !actor.IsAdmin() {
		return model.PublicUser{}, model.ErrForbidden
	}
	role, ok := model.ParseUserRole(req.Role)
	if !ok || role == "" {
		return model.PublicUser{}, model.ErrInvalidRole
	}
	username := strings.ToLower(validate.Trim(req.Username))
	if !validate.UsernameOK(username) {
		return model.PublicUser{}, model.ErrInvalidUsername
	}
	if !validate.PasswordOK(req.Password) {
		return model.PublicUser{}, model.ErrInvalidPassword
	}
	display := validate.SanitizePlain(req.DisplayName)
	if !validate.InRange(display, 1, policy.DisplayNameMax) {
		return model.PublicUser{}, model.ErrInvalidDisplayName
	}
	if _, err := s.store.GetUserByUsername(ctx, username); err == nil {
		return model.PublicUser{}, model.ErrAlreadyExists
	}
	salt, hash, it, err := s.hasher.Hash(req.Password)
	if err != nil {
		return model.PublicUser{}, err
	}
	now := s.clock.Now()
	u, err := s.store.CreateUser(ctx, model.User{
		Username:     username,
		DisplayName:  display,
		Role:         role,
		Status:       model.UserActive,
		PasswordSalt: salt,
		PasswordHash: hash,
		Iterations:   it,
		Phone:        validate.Trim(req.Phone),
		CreatedAt:    now,
		UpdatedAt:    now,
	})
	if err != nil {
		return model.PublicUser{}, err
	}
	return u.Public(), nil
}

func (s *UserService) SetStatus(ctx context.Context, actor model.User, id string, status model.UserStatus) (model.PublicUser, error) {
	if !actor.IsAdmin() {
		return model.PublicUser{}, model.ErrForbidden
	}
	if status != model.UserActive && status != model.UserFrozen && status != model.UserBanned {
		return model.PublicUser{}, model.ErrInvalidUserStatus
	}
	u, err := s.store.GetUserByID(ctx, id)
	if err != nil {
		return model.PublicUser{}, err
	}
	if u.IsAdmin() && u.ID != actor.ID {
		return model.PublicUser{}, model.ErrForbidden
	}
	u.Status = status
	u.UpdatedAt = s.clock.Now()
	u, err = s.store.UpdateUser(ctx, u)
	if err != nil {
		return model.PublicUser{}, err
	}
	if status != model.UserActive {
		s.sessions.InvalidateByUser(u.ID)
	}
	return u.Public(), nil
}

func (s *UserService) Notifications(ctx context.Context, actor model.User, unreadOnly bool) ([]model.Notification, error) {
	return s.store.ListNotifications(ctx, actor.ID, unreadOnly)
}

func (s *UserService) ReadNotification(ctx context.Context, actor model.User, id string) error {
	n, err := s.store.GetNotification(ctx, id)
	if err != nil {
		return err
	}
	if n.UserID != actor.ID {
		return model.ErrForbidden
	}
	if n.Read {
		return nil
	}
	n.Read = true
	now := s.clock.Now()
	n.ReadAt = &now
	_, err = s.store.UpdateNotification(ctx, n)
	return err
}

func (s *UserService) ReadAll(ctx context.Context, actor model.User) (int, error) {
	return s.store.MarkAllNotificationsRead(ctx, actor.ID, s.clock.Now())
}

func (s *UserService) HourLedgers(ctx context.Context, actor model.User) ([]model.HourLedger, error) {
	return s.store.ListHourLedgers(ctx, actor.ID)
}

func (s *UserService) PointLedgers(ctx context.Context, actor model.User) ([]model.PointLedger, error) {
	return s.store.ListPointLedgers(ctx, actor.ID)
}
