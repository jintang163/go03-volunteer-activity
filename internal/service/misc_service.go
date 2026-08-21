package service

import (
	"context"

	"go03-volunteer-activity/internal/model"
	"go03-volunteer-activity/internal/store"
	"go03-volunteer-activity/internal/validate"
)

type NotifyService struct {
	store store.Store
	clock Clock
}

func NewNotifyService(s store.Store, clock Clock) *NotifyService {
	return &NotifyService{store: s, clock: clock}
}

func (s *NotifyService) Push(ctx context.Context, userID string, kind model.NotificationKind, title, body, related string) error {
	_, err := s.store.CreateNotification(ctx, model.Notification{
		UserID:    userID,
		Kind:      kind,
		Title:     title,
		Body:      body,
		RelatedID: related,
		CreatedAt: s.clock.Now(),
	})
	return err
}

type AuditService struct {
	store store.Store
	clock Clock
}

func NewAuditService(s store.Store, clock Clock) *AuditService {
	return &AuditService{store: s, clock: clock}
}

func (s *AuditService) Log(ctx context.Context, actorID string, action model.AuditAction, targetType, targetID, detail string) error {
	_, err := s.store.CreateAudit(ctx, model.AuditLog{
		ActorID:    actorID,
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		Detail:     detail,
		CreatedAt:  s.clock.Now(),
	})
	return err
}

func (s *AuditService) List(ctx context.Context, actor model.User, targetType, targetID string) ([]model.AuditLog, error) {
	if !actor.IsAdmin() {
		return nil, model.ErrForbidden
	}
	return s.store.ListAudits(ctx, targetType, targetID)
}

type TeamService struct {
	store store.Store
	clock Clock
}

func NewTeamService(s store.Store, clock Clock) *TeamService {
	return &TeamService{store: s, clock: clock}
}

func (s *TeamService) Create(ctx context.Context, actor model.User, req model.CreateTeamRequest) (model.Team, error) {
	if err := requireActiveWriter(actor); err != nil {
		return model.Team{}, err
	}
	if !actor.IsOrganizer() {
		return model.Team{}, model.ErrForbidden
	}
	name := validate.SanitizePlain(req.Name)
	if !validate.InRange(name, 1, 40) {
		return model.Team{}, model.ErrValidation
	}
	now := s.clock.Now()
	t, err := s.store.CreateTeam(ctx, model.Team{
		OwnerID:     actor.ID,
		Name:        name,
		Description: validate.SanitizePlain(req.Description),
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		return model.Team{}, err
	}
	_, err = s.store.CreateTeamMember(ctx, model.TeamMember{
		TeamID:   t.ID,
		UserID:   actor.ID,
		Role:     "owner",
		JoinedAt: now,
	})
	return t, err
}

func (s *TeamService) ListMine(ctx context.Context, actor model.User) ([]model.Team, error) {
	all, err := s.store.ListTeams(ctx, "")
	if err != nil {
		return nil, err
	}
	out := []model.Team{}
	for _, t := range all {
		if t.OwnerID == actor.ID {
			out = append(out, t)
			continue
		}
		if _, err := s.store.GetTeamMember(ctx, t.ID, actor.ID); err == nil {
			out = append(out, t)
		}
	}
	return out, nil
}

func (s *TeamService) Invite(ctx context.Context, actor model.User, teamID string, username string) (model.TeamMember, error) {
	t, err := s.store.GetTeam(ctx, teamID)
	if err != nil {
		return model.TeamMember{}, err
	}
	if t.OwnerID != actor.ID && !actor.IsAdmin() {
		return model.TeamMember{}, model.ErrNotTeamOwner
	}
	u, err := s.store.GetUserByUsername(ctx, username)
	if err != nil {
		return model.TeamMember{}, err
	}
	if _, err := s.store.GetTeamMember(ctx, teamID, u.ID); err == nil {
		return model.TeamMember{}, model.ErrAlreadyTeamMember
	}
	return s.store.CreateTeamMember(ctx, model.TeamMember{
		TeamID:   teamID,
		UserID:   u.ID,
		Role:     "member",
		JoinedAt: s.clock.Now(),
	})
}

func (s *TeamService) Members(ctx context.Context, actor model.User, teamID string) ([]model.TeamMember, error) {
	if _, err := s.store.GetTeamMember(ctx, teamID, actor.ID); err != nil && !actor.IsAdmin() {
		t, err2 := s.store.GetTeam(ctx, teamID)
		if err2 != nil {
			return nil, err2
		}
		if t.OwnerID != actor.ID {
			return nil, model.ErrForbidden
		}
	}
	return s.store.ListTeamMembers(ctx, teamID)
}

type CertService struct {
	store  store.Store
	notify *NotifyService
	audit  *AuditService
	clock  Clock
}

func NewCertService(s store.Store, notify *NotifyService, audit *AuditService, clock Clock) *CertService {
	return &CertService{store: s, notify: notify, audit: audit, clock: clock}
}

func (s *CertService) MaybeIssue(ctx context.Context, actor model.User, volunteer model.User) error {
	tier := model.CertificateTier("")
	mins := volunteer.TotalMinutes
	switch {
	case mins >= 100*60:
		tier = model.CertGold
	case mins >= 50*60:
		tier = model.CertSilver
	case mins >= 20*60:
		tier = model.CertBronze
	default:
		return nil
	}
	ok, err := s.store.HasCertificateTier(ctx, volunteer.ID, tier)
	if err != nil || ok {
		return err
	}
	c, err := s.store.CreateCertificate(ctx, model.Certificate{
		UserID:       volunteer.ID,
		Tier:         tier,
		HoursAtIssue: mins / 60,
		IssuedAt:     s.clock.Now(),
		IssuerID:     actor.ID,
	})
	if err != nil {
		return err
	}
	_ = s.audit.Log(ctx, actor.ID, model.AuditIssueCert, "certificate", c.ID, string(tier))
	_ = s.notify.Push(ctx, volunteer.ID, model.NotifyCertificate, "获得志愿证书", "档位 "+string(tier)+"，核验码 "+c.Code, c.ID)
	return nil
}

func (s *CertService) ListMine(ctx context.Context, actor model.User) ([]model.Certificate, error) {
	return s.store.ListCertificatesByUser(ctx, actor.ID)
}

func (s *CertService) Verify(ctx context.Context, code string) (model.Certificate, model.PublicUser, error) {
	c, err := s.store.GetCertificateByCode(ctx, validate.Trim(code))
	if err != nil {
		return model.Certificate{}, model.PublicUser{}, err
	}
	u, err := publicOf(ctx, s.store, c.UserID)
	return c, u, err
}

type StatsService struct {
	store store.Store
	clock Clock
}

func NewStatsService(s store.Store, clock Clock) *StatsService {
	return &StatsService{store: s, clock: clock}
}

func (s *StatsService) Global(ctx context.Context, actor model.User) (model.StatsSnapshot, error) {
	if !actor.IsAdmin() {
		return model.StatsSnapshot{}, model.ErrForbidden
	}
	total, active, _, err := s.store.CountUsers(ctx)
	if err != nil {
		return model.StatsSnapshot{}, err
	}
	byStatus, err := s.store.CountActivitiesByStatus(ctx)
	if err != nil {
		return model.StatsSnapshot{}, err
	}
	m := map[string]int{}
	for k, v := range byStatus {
		m[string(k)] = v
	}
	acts, _ := s.store.ListActivities(ctx, model.ActivityFilter{IncludeDraft: true})
	approved, wait, noshow := 0, 0, 0
	for _, a := range acts {
		sgs, _ := s.store.ListSignupsByActivity(ctx, a.ID)
		for _, sg := range sgs {
			switch sg.Status {
			case model.SignupApproved:
				approved++
			case model.SignupWaitlisted:
				wait++
			case model.SignupNoShow:
				noshow++
			}
		}
	}
	rate := 0
	if approved+noshow > 0 {
		rate = noshow * 10000 / (approved + noshow)
	}
	pending, _ := s.store.ListPendingHours(ctx)
	now := s.clock.Now()
	todayCI, _ := s.store.CountCheckInsOnDay(ctx, now)
	monthMin := 0
	users, _ := s.store.ListUsers(ctx, model.UserFilter{})
	for _, u := range users {
		leds, _ := s.store.ListHourLedgers(ctx, u.ID)
		for _, l := range leds {
			if l.CreatedAt.Year() == now.Year() && l.CreatedAt.Month() == now.Month() {
				monthMin += l.DeltaMin
			}
		}
	}
	return model.StatsSnapshot{
		UsersTotal:         total,
		UsersActive:        active,
		ActivitiesByStatus: m,
		SignupsApproved:    approved,
		SignupsWaitlist:    wait,
		CheckInsToday:      todayCI,
		HoursThisMonth:     monthMin,
		NoShowRateBP:       rate,
		PendingHours:       len(pending),
	}, nil
}

func (s *StatsService) Organizer(ctx context.Context, actor model.User) (model.OrganizerStats, error) {
	if !actor.IsOrganizer() {
		return model.OrganizerStats{}, model.ErrForbidden
	}
	acts, err := s.store.ListActivities(ctx, model.ActivityFilter{OrganizerID: actor.ID, IncludeDraft: true})
	if err != nil {
		return model.OrganizerStats{}, err
	}
	st := model.OrganizerStats{MyActivities: len(acts)}
	for _, a := range acts {
		sgs, _ := s.store.ListSignupsByActivity(ctx, a.ID)
		for _, sg := range sgs {
			if sg.Status == model.SignupApproved {
				st.ApprovedPeople++
			}
		}
		cis, _ := s.store.ListCheckInsByActivity(ctx, a.ID)
		st.CheckedIn += len(cis)
		hrs, _ := s.store.ListHoursByActivity(ctx, a.ID)
		for _, h := range hrs {
			if h.Status == model.HourPending || h.Status == model.HourDraft {
				st.PendingHours++
			}
		}
	}
	return st, nil
}
