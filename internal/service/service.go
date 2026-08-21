package service

import (
	"context"
	"time"

	"go03-volunteer-activity/internal/auth"
	"go03-volunteer-activity/internal/model"
	"go03-volunteer-activity/internal/store"
)

type Clock interface {
	Now() time.Time
}

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now() }

var DefaultClock Clock = systemClock{}

type Services struct {
	Auth     *AuthService
	User     *UserService
	Activity *ActivityService
	Signup   *SignupService
	CheckIn  *CheckInService
	Hours    *HoursService
	Notify   *NotifyService
	Team     *TeamService
	Stats    *StatsService
	Cert     *CertService
	Audit    *AuditService
}

func NewServices(s store.Store, hasher *auth.PasswordHasher, sessions *auth.SessionManager, clock Clock, noShowLimit int) *Services {
	if clock == nil {
		clock = DefaultClock
	}
	notify := NewNotifyService(s, clock)
	audit := NewAuditService(s, clock)
	svc := &Services{Notify: notify, Audit: audit}
	svc.Auth = NewAuthService(s, hasher, sessions, clock, notify)
	svc.User = NewUserService(s, hasher, sessions, clock)
	svc.Activity = NewActivityService(s, notify, audit, clock)
	svc.Signup = NewSignupService(s, notify, audit, clock, noShowLimit)
	svc.CheckIn = NewCheckInService(s, notify, audit, clock)
	svc.Hours = NewHoursService(s, notify, audit, clock)
	svc.Team = NewTeamService(s, clock)
	svc.Stats = NewStatsService(s, clock)
	svc.Cert = NewCertService(s, notify, audit, clock)
	return svc
}

type ctxKey string

const ctxUserKey ctxKey = "user"

func WithUser(ctx context.Context, u model.User) context.Context {
	return context.WithValue(ctx, ctxUserKey, u)
}

func UserFromContext(ctx context.Context) (model.User, bool) {
	u, ok := ctx.Value(ctxUserKey).(model.User)
	return u, ok
}

func MustUserFromContext(ctx context.Context) model.User {
	u, ok := UserFromContext(ctx)
	if !ok {
		panic("service: user not found in context")
	}
	return u
}

func requireActiveWriter(u model.User) error {
	if u.IsAdmin() {
		return nil
	}
	return u.CanWrite()
}

func publicOf(ctx context.Context, s store.Store, id string) (model.PublicUser, error) {
	u, err := s.GetUserByID(ctx, id)
	if err != nil {
		return model.PublicUser{}, err
	}
	return u.Public(), nil
}

func canManageActivity(actor model.User, act model.Activity) bool {
	if actor.IsAdmin() {
		return true
	}
	return actor.ID == act.OrganizerID && actor.IsOrganizer()
}
