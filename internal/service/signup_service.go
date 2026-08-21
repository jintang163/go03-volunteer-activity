package service

import (
	"context"
	"time"

	"go03-volunteer-activity/internal/model"
	"go03-volunteer-activity/internal/policy"
	"go03-volunteer-activity/internal/store"
	"go03-volunteer-activity/internal/validate"
)

type SignupService struct {
	store       store.Store
	notify      *NotifyService
	audit       *AuditService
	clock       Clock
	noShowLimit int
}

func NewSignupService(s store.Store, notify *NotifyService, audit *AuditService, clock Clock, noShowLimit int) *SignupService {
	if noShowLimit <= 0 {
		noShowLimit = policy.DefaultNoShowLimit
	}
	return &SignupService{store: s, notify: notify, audit: audit, clock: clock, noShowLimit: noShowLimit}
}

func (s *SignupService) Signup(ctx context.Context, actor model.User, activityID string, req model.SignupRequest) (model.Signup, error) {
	if err := requireActiveWriter(actor); err != nil {
		return model.Signup{}, err
	}
	if !actor.IsVolunteer() && !actor.IsAdmin() {
		return model.Signup{}, model.ErrNotVolunteer
	}
	act, err := s.store.GetActivity(ctx, activityID)
	if err != nil {
		return model.Signup{}, err
	}
	if actor.ID == act.OrganizerID {
		return model.Signup{}, model.ErrCannotSignupOwn
	}
	if !act.Status.AllowsSignup() {
		return model.Signup{}, model.ErrActivityNotOpen
	}
	now := s.clock.Now()
	if now.Before(act.SignupOpenAt) {
		return model.Signup{}, model.ErrSignupWindowNotOpen
	}
	if !now.Before(act.SignupCloseAt) {
		return model.Signup{}, model.ErrSignupWindowClosed
	}
	if act.MinAge > 0 && actor.Age > 0 && actor.Age < act.MinAge {
		return model.Signup{}, model.ErrAgeRequirement
	}
	if !actor.HasAllSkills(act.RequiredSkills) {
		return model.Signup{}, model.ErrSkillMismatch
	}
	if act.TeamID != "" {
		if _, err := s.store.GetTeamMember(ctx, act.TeamID, actor.ID); err != nil {
			return model.Signup{}, model.ErrTeamOnly
		}
	}
	since := now.Add(-time.Duration(policy.NoShowWindowDays) * 24 * time.Hour)
	ns, err := s.store.CountNoShowsSince(ctx, actor.ID, since)
	if err != nil {
		return model.Signup{}, err
	}
	if ns >= s.noShowLimit {
		return model.Signup{}, model.ErrTooManyNoShows
	}
	if existing, err := s.store.GetSignupByActivityVolunteer(ctx, activityID, actor.ID); err == nil && existing.Status.IsActive() {
		return model.Signup{}, model.ErrAlreadySignedUp
	}

	// 原子地完成「容量/候补/冲突/审批判定 + 写入」。此前采用先 CountApprovedByActivity
	// 检查、后 CreateSignup 写入的两步式流程，两次加锁之间存在 TOCTOU 竞态：并发请求
	// 都能读到 approved < capacity 而判定录取，最终保存多条已录取记录造成超额。现把
	// 整个决策与写入下沉到 store.ReserveSignup 的单次写锁内，同一时刻至多一个请求成功。
	sg := model.Signup{
		ActivityID:  activityID,
		VolunteerID: actor.ID,
		Remark:      validate.SanitizePlain(req.Remark),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	saved, err := s.store.ReserveSignup(ctx, sg, act)
	if err != nil {
		return model.Signup{}, err
	}
	_, _ = s.store.RecountActivityCounters(ctx, act.ID)
	msg := "报名已提交"
	switch saved.Status {
	case model.SignupApproved:
		msg = "你已录取「" + act.Title + "」"
	case model.SignupWaitlisted:
		msg = "活动已满员，你已进入候补名单"
	case model.SignupPending:
		msg = "报名待组织者审核"
	}
	_ = s.notify.Push(ctx, actor.ID, model.NotifySignupResult, "报名结果", msg, act.ID)
	return saved, nil
}

func (s *SignupService) Approve(ctx context.Context, actor model.User, signupID string) (model.Signup, error) {
	sg, act, err := s.loadManaged(ctx, actor, signupID)
	if err != nil {
		return model.Signup{}, err
	}
	if sg.Status != model.SignupPending && sg.Status != model.SignupWaitlisted {
		return model.Signup{}, model.ErrConflict
	}
	approved, err := s.store.CountApprovedByActivity(ctx, act.ID)
	if err != nil {
		return model.Signup{}, err
	}
	if approved >= act.Capacity {
		return model.Signup{}, model.ErrCapacityFull
	}
	overlap, err := s.store.ListApprovedOverlapping(ctx, sg.VolunteerID, act.StartAt, act.EndAt, act.ID)
	if err != nil {
		return model.Signup{}, err
	}
	if len(overlap) > 0 {
		return model.Signup{}, model.ErrScheduleConflict
	}
	now := s.clock.Now()
	sg.Status = model.SignupApproved
	sg.WaitlistSeq = 0
	sg.ApprovedAt = &now
	sg.UpdatedAt = now
	saved, err := s.store.UpdateSignup(ctx, sg)
	if err != nil {
		return model.Signup{}, err
	}
	_, _ = s.store.RecountActivityCounters(ctx, act.ID)
	_ = s.audit.Log(ctx, actor.ID, model.AuditApproveSignup, "signup", saved.ID, act.Title)
	_ = s.notify.Push(ctx, sg.VolunteerID, model.NotifySignupResult, "报名已录取", "你已录取「"+act.Title+"」", act.ID)
	return saved, nil
}

func (s *SignupService) Reject(ctx context.Context, actor model.User, signupID string, reason string) (model.Signup, error) {
	sg, act, err := s.loadManaged(ctx, actor, signupID)
	if err != nil {
		return model.Signup{}, err
	}
	if !sg.Status.IsActive() {
		return model.Signup{}, model.ErrConflict
	}
	wasApproved := sg.Status == model.SignupApproved
	now := s.clock.Now()
	sg.Status = model.SignupRejected
	sg.RejectReason = validate.SanitizePlain(reason)
	sg.UpdatedAt = now
	saved, err := s.store.UpdateSignup(ctx, sg)
	if err != nil {
		return model.Signup{}, err
	}
	_, _ = s.store.RecountActivityCounters(ctx, act.ID)
	_ = s.audit.Log(ctx, actor.ID, model.AuditRejectSignup, "signup", saved.ID, sg.RejectReason)
	_ = s.notify.Push(ctx, sg.VolunteerID, model.NotifySignupResult, "报名未通过", "「"+act.Title+"」："+sg.RejectReason, act.ID)
	if wasApproved {
		_ = s.promoteWaitlist(ctx, act)
	}
	return saved, nil
}

func (s *SignupService) Cancel(ctx context.Context, actor model.User, signupID string) (model.Signup, error) {
	sg, err := s.store.GetSignup(ctx, signupID)
	if err != nil {
		return model.Signup{}, err
	}
	act, err := s.store.GetActivity(ctx, sg.ActivityID)
	if err != nil {
		return model.Signup{}, err
	}
	if sg.VolunteerID != actor.ID && !canManageActivity(actor, act) {
		return model.Signup{}, model.ErrForbidden
	}
	if !sg.Status.IsActive() {
		return model.Signup{}, model.ErrConflict
	}
	now := s.clock.Now()
	wasApproved := sg.Status == model.SignupApproved
	late := wasApproved && !now.Before(act.StartAt)
	sg.Status = model.SignupCancelled
	sg.CancelledAt = &now
	sg.UpdatedAt = now
	saved, err := s.store.UpdateSignup(ctx, sg)
	if err != nil {
		return model.Signup{}, err
	}
	_, _ = s.store.RecountActivityCounters(ctx, act.ID)
	if late {
		u, err := s.store.GetUserByID(ctx, sg.VolunteerID)
		if err == nil {
			_, _ = s.store.ApplyPoints(ctx, u, model.PointLedger{
				UserID:    u.ID,
				Delta:     -policy.LateCancelPointPenalty,
				Reason:    model.PointLateCancel,
				RelatedID: act.ID,
				Note:      "活动开始后取消报名",
				CreatedAt: now,
			})
		}
	}
	if wasApproved {
		_ = s.promoteWaitlist(ctx, act)
	}
	return saved, nil
}

func (s *SignupService) promoteWaitlist(ctx context.Context, act model.Activity) error {
	if act.Status == model.ActivityCancelled || act.Status == model.ActivityCompleted {
		return nil
	}
	list, err := s.store.ListSignupsByActivity(ctx, act.ID)
	if err != nil {
		return err
	}
	var best *model.Signup
	for i := range list {
		sg := list[i]
		if sg.Status != model.SignupWaitlisted {
			continue
		}
		if best == nil || sg.WaitlistSeq < best.WaitlistSeq {
			cp := sg
			best = &cp
		}
	}
	if best == nil {
		return nil
	}
	approved, err := s.store.CountApprovedByActivity(ctx, act.ID)
	if err != nil {
		return err
	}
	if approved >= act.Capacity {
		return nil
	}
	overlap, err := s.store.ListApprovedOverlapping(ctx, best.VolunteerID, act.StartAt, act.EndAt, act.ID)
	if err != nil {
		return err
	}
	now := s.clock.Now()
	if len(overlap) > 0 {
		best.Status = model.SignupRejected
		best.RejectReason = "时间冲突，自动跳过候补"
		best.UpdatedAt = now
		if _, err := s.store.UpdateSignup(ctx, *best); err != nil {
			return err
		}
		return s.promoteWaitlist(ctx, act)
	}
	best.Status = model.SignupApproved
	best.WaitlistSeq = 0
	best.ApprovedAt = &now
	best.UpdatedAt = now
	if _, err := s.store.UpdateSignup(ctx, *best); err != nil {
		return err
	}
	_, _ = s.store.RecountActivityCounters(ctx, act.ID)
	_ = s.notify.Push(ctx, best.VolunteerID, model.NotifyWaitlistUp, "候补递补成功", "你已从候补录取「"+act.Title+"」", act.ID)
	return nil
}

func (s *SignupService) ListByActivity(ctx context.Context, actor model.User, activityID string) ([]model.PublicSignup, error) {
	act, err := s.store.GetActivity(ctx, activityID)
	if err != nil {
		return nil, err
	}
	if !canManageActivity(actor, act) {
		return nil, model.ErrNotOrganizer
	}
	list, err := s.store.ListSignupsByActivity(ctx, activityID)
	if err != nil {
		return nil, err
	}
	out := make([]model.PublicSignup, 0, len(list))
	for _, sg := range list {
		pu, err := publicOf(ctx, s.store, sg.VolunteerID)
		if err != nil {
			pu = model.PublicUser{ID: sg.VolunteerID}
		}
		out = append(out, model.PublicSignup{Signup: sg, Volunteer: pu})
	}
	return out, nil
}

func (s *SignupService) Mine(ctx context.Context, actor model.User) ([]model.Signup, error) {
	return s.store.ListSignupsByVolunteer(ctx, actor.ID)
}

func (s *SignupService) loadManaged(ctx context.Context, actor model.User, signupID string) (model.Signup, model.Activity, error) {
	sg, err := s.store.GetSignup(ctx, signupID)
	if err != nil {
		return model.Signup{}, model.Activity{}, err
	}
	act, err := s.store.GetActivity(ctx, sg.ActivityID)
	if err != nil {
		return model.Signup{}, model.Activity{}, err
	}
	if !canManageActivity(actor, act) {
		return model.Signup{}, model.Activity{}, model.ErrNotOrganizer
	}
	if !act.Status.AllowsManageRoster() && act.Status != model.ActivityDraft {
		return model.Signup{}, model.Activity{}, model.ErrInvalidActivityStatus
	}
	return sg, act, nil
}
