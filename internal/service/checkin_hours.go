package service

import (
	"context"
	"strings"
	"time"

	"go03-volunteer-activity/internal/hoursutil"
	"go03-volunteer-activity/internal/model"
	"go03-volunteer-activity/internal/policy"
	"go03-volunteer-activity/internal/store"
	"go03-volunteer-activity/internal/validate"
)

type CheckInService struct {
	store  store.Store
	notify *NotifyService
	audit  *AuditService
	clock  Clock
}

func NewCheckInService(s store.Store, notify *NotifyService, audit *AuditService, clock Clock) *CheckInService {
	return &CheckInService{store: s, notify: notify, audit: audit, clock: clock}
}

func (s *CheckInService) SelfCheckIn(ctx context.Context, actor model.User, activityID string, req model.CheckInRequest) (model.CheckIn, error) {
	act, sg, err := s.requireApproved(ctx, actor.ID, activityID)
	if err != nil {
		return model.CheckIn{}, err
	}
	if !act.Status.AllowsCheckIn() {
		return model.CheckIn{}, model.ErrInvalidActivityStatus
	}
	now := s.clock.Now()
	if !hoursutil.InCheckInWindow(now, act.StartAt, act.EndAt, act.CheckInOpenBefore, act.CheckOutGrace) {
		return model.CheckIn{}, model.ErrCheckInWindowClosed
	}
	code := strings.ToUpper(validate.Trim(req.Code))
	if code == "" || code != strings.ToUpper(act.CheckInCode) {
		return model.CheckIn{}, model.ErrWrongCheckInCode
	}
	if _, err := s.store.GetCheckInByActivityVolunteer(ctx, activityID, actor.ID); err == nil {
		return model.CheckIn{}, model.ErrAlreadyCheckedIn
	}
	rec := model.CheckIn{
		ActivityID:  activityID,
		VolunteerID: actor.ID,
		SignupID:    sg.ID,
		Method:      model.CheckInCode,
		CheckInAt:   now,
		ActorID:     actor.ID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	// 原子地完成「检查是否已签到 + 写入」。此前采用先 GetCheckInByActivityVolunteer 检查、
	// 后 CreateCheckIn 写入的两步式流程，两次加锁之间存在 TOCTOU 竞态：同一志愿者并发提交
	// 两次正确签到码时，两个请求都能通过存在性检查而各自写入，最终保存两条签到记录。现把
	// 检查与写入下沉到 store.ReserveCheckIn 的单次写锁内，第二个请求在锁内重新读到已存在的
	// 签到记录即返回 ErrAlreadyCheckedIn，保证同一志愿者在同一活动只能签到一次。
	saved, err := s.store.ReserveCheckIn(ctx, rec)
	if err != nil {
		return model.CheckIn{}, err
	}
	_ = s.audit.Log(ctx, actor.ID, model.AuditCheckIn, "checkin", saved.ID, act.Title)
	return saved, nil
}

func (s *CheckInService) CheckOut(ctx context.Context, actor model.User, activityID string) (model.CheckIn, error) {
	act, _, err := s.requireApproved(ctx, actor.ID, activityID)
	if err != nil {
		return model.CheckIn{}, err
	}
	ci, err := s.store.GetCheckInByActivityVolunteer(ctx, activityID, actor.ID)
	if err != nil {
		return model.CheckIn{}, model.ErrNotCheckedIn
	}
	if ci.HasCheckedOut() {
		return model.CheckIn{}, model.ErrAlreadyCheckedOut
	}
	now := s.clock.Now()
	if !hoursutil.InCheckOutWindow(now, act.StartAt, act.EndAt, act.CheckOutGrace) {
		return model.CheckIn{}, model.ErrCheckOutWindowClosed
	}
	ci.CheckOutAt = &now
	ci.UpdatedAt = now
	saved, err := s.store.UpdateCheckIn(ctx, ci)
	if err != nil {
		return model.CheckIn{}, err
	}
	_ = s.audit.Log(ctx, actor.ID, model.AuditCheckOut, "checkin", saved.ID, act.Title)
	_, _ = s.autoDraftHours(ctx, act, saved)
	return saved, nil
}

func (s *CheckInService) ProxyCheckIn(ctx context.Context, actor model.User, activityID string, req model.ProxyCheckInRequest) (model.CheckIn, error) {
	act, err := s.store.GetActivity(ctx, activityID)
	if err != nil {
		return model.CheckIn{}, err
	}
	if !canManageActivity(actor, act) {
		return model.CheckIn{}, model.ErrNotOrganizer
	}
	sg, err := s.store.GetSignupByActivityVolunteer(ctx, activityID, req.VolunteerID)
	if err != nil {
		return model.CheckIn{}, err
	}
	if sg.Status != model.SignupApproved {
		return model.CheckIn{}, model.ErrNotApprovedSignup
	}
	if _, err := s.store.GetCheckInByActivityVolunteer(ctx, activityID, req.VolunteerID); err == nil {
		return model.CheckIn{}, model.ErrAlreadyCheckedIn
	}
	now := s.clock.Now()
	at := now
	if req.CheckInAt != nil && !req.CheckInAt.IsZero() {
		at = *req.CheckInAt
	}
	open := act.StartAt.Add(-time.Duration(act.CheckInOpenBefore) * time.Minute)
	if at.Before(open) {
		return model.CheckIn{}, model.ErrCheckInWindowClosed
	}
	if now.Sub(at) > policy.ProxyCheckInMaxLate {
		return model.CheckIn{}, model.ErrCheckInWindowClosed
	}
	rec := model.CheckIn{
		ActivityID:  activityID,
		VolunteerID: req.VolunteerID,
		SignupID:    sg.ID,
		Method:      model.CheckInOrganizer,
		CheckInAt:   at,
		Note:        validate.SanitizePlain(req.Note),
		ActorID:     actor.ID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	// 同 SelfCheckIn：把存在性检查与写入合并进 store.ReserveCheckIn 的单次写锁，
	// 消除「先 GetCheckInByActivityVolunteer、后 CreateCheckIn」之间的 TOCTOU 竞态。
	saved, err := s.store.ReserveCheckIn(ctx, rec)
	if err != nil {
		return model.CheckIn{}, err
	}
	_ = s.audit.Log(ctx, actor.ID, model.AuditCheckIn, "checkin", saved.ID, "proxy")
	return saved, nil
}

func (s *CheckInService) List(ctx context.Context, actor model.User, activityID string) ([]model.CheckIn, error) {
	act, err := s.store.GetActivity(ctx, activityID)
	if err != nil {
		return nil, err
	}
	if !canManageActivity(actor, act) {
		return nil, model.ErrNotOrganizer
	}
	return s.store.ListCheckInsByActivity(ctx, activityID)
}

func (s *CheckInService) requireApproved(ctx context.Context, volunteerID, activityID string) (model.Activity, model.Signup, error) {
	act, err := s.store.GetActivity(ctx, activityID)
	if err != nil {
		return model.Activity{}, model.Signup{}, err
	}
	sg, err := s.store.GetSignupByActivityVolunteer(ctx, activityID, volunteerID)
	if err != nil {
		return model.Activity{}, model.Signup{}, err
	}
	if sg.Status != model.SignupApproved {
		return model.Activity{}, model.Signup{}, model.ErrNotApprovedSignup
	}
	return act, sg, nil
}

func (s *CheckInService) autoDraftHours(ctx context.Context, act model.Activity, ci model.CheckIn) (model.HourRecord, error) {
	if !ci.HasCheckedOut() {
		return model.HourRecord{}, model.ErrNoCheckoutForAutoHours
	}
	if existing, err := s.store.GetOpenHour(ctx, act.ID, ci.VolunteerID); err == nil {
		if existing.Status == model.HourApproved {
			return existing, nil
		}
		raw := hoursutil.RawMinutes(ci.CheckInAt, *ci.CheckOutAt)
		existing.RawMinutes = raw
		existing.WorkMinutes = hoursutil.CapMinutes(raw, existing.BreakMinutes, act.PlannedMinutes)
		existing.CheckInID = ci.ID
		existing.UpdatedAt = s.clock.Now()
		if existing.Status == model.HourDraft {
			return s.store.UpdateHour(ctx, existing)
		}
		return existing, nil
	}
	raw := hoursutil.RawMinutes(ci.CheckInAt, *ci.CheckOutAt)
	work := hoursutil.CapMinutes(raw, 0, act.PlannedMinutes)
	now := s.clock.Now()
	return s.store.CreateHour(ctx, model.HourRecord{
		ActivityID:   act.ID,
		VolunteerID:  ci.VolunteerID,
		SignupID:     ci.SignupID,
		CheckInID:    ci.ID,
		Source:       model.HourSourceAuto,
		Status:       model.HourDraft,
		RawMinutes:   raw,
		WorkMinutes:  work,
		CreatedAt:    now,
		UpdatedAt:    now,
	})
}

type HoursService struct {
	store  store.Store
	notify *NotifyService
	audit  *AuditService
	clock  Clock
	cert   *CertService
}

func NewHoursService(s store.Store, notify *NotifyService, audit *AuditService, clock Clock) *HoursService {
	return &HoursService{store: s, notify: notify, audit: audit, clock: clock, cert: NewCertService(s, notify, audit, clock)}
}

func (s *HoursService) SubmitManual(ctx context.Context, actor model.User, req model.SubmitHoursRequest) (model.HourRecord, error) {
	act, err := s.store.GetActivity(ctx, req.ActivityID)
	if err != nil {
		return model.HourRecord{}, err
	}
	if !canManageActivity(actor, act) {
		return model.HourRecord{}, model.ErrNotOrganizer
	}
	if req.WorkMinutes <= 0 || req.WorkMinutes > policy.MaxPlannedMinutes {
		return model.HourRecord{}, model.ErrInvalidMinutes
	}
	sg, err := s.store.GetSignupByActivityVolunteer(ctx, req.ActivityID, req.VolunteerID)
	if err != nil {
		return model.HourRecord{}, err
	}
	if sg.Status != model.SignupApproved && sg.Status != model.SignupNoShow {
		return model.HourRecord{}, model.ErrNotApprovedSignup
	}
	if existing, err := s.store.GetOpenHour(ctx, req.ActivityID, req.VolunteerID); err == nil {
		if existing.Status == model.HourApproved || existing.Status == model.HourPending {
			return model.HourRecord{}, model.ErrHoursAlreadyExists
		}
	}
	work := hoursutil.CapMinutes(req.WorkMinutes+req.BreakMinutes, req.BreakMinutes, act.PlannedMinutes)
	now := s.clock.Now()
	return s.store.CreateHour(ctx, model.HourRecord{
		ActivityID:   req.ActivityID,
		VolunteerID:  req.VolunteerID,
		SignupID:     sg.ID,
		Source:       model.HourSourceManual,
		Status:       model.HourPending,
		RawMinutes:   req.WorkMinutes + req.BreakMinutes,
		BreakMinutes: req.BreakMinutes,
		WorkMinutes:  work,
		Note:         validate.SanitizePlain(req.Note),
		CreatedAt:    now,
		UpdatedAt:    now,
	})
}

func (s *HoursService) SubmitAutoForReview(ctx context.Context, actor model.User, hourID string, breakMinutes int) (model.HourRecord, error) {
	h, err := s.store.GetHour(ctx, hourID)
	if err != nil {
		return model.HourRecord{}, err
	}
	act, err := s.store.GetActivity(ctx, h.ActivityID)
	if err != nil {
		return model.HourRecord{}, err
	}
	if !canManageActivity(actor, act) {
		return model.HourRecord{}, model.ErrNotOrganizer
	}
	if h.Status != model.HourDraft {
		return model.HourRecord{}, model.ErrHoursNotPending
	}
	h.BreakMinutes = breakMinutes
	h.WorkMinutes = hoursutil.CapMinutes(h.RawMinutes, breakMinutes, act.PlannedMinutes)
	h.Status = model.HourPending
	h.UpdatedAt = s.clock.Now()
	return s.store.UpdateHour(ctx, h)
}

func (s *HoursService) Approve(ctx context.Context, actor model.User, hourID string, req model.ReviewHoursRequest) (model.HourRecord, error) {
	h, err := s.store.GetHour(ctx, hourID)
	if err != nil {
		return model.HourRecord{}, err
	}
	act, err := s.store.GetActivity(ctx, h.ActivityID)
	if err != nil {
		return model.HourRecord{}, err
	}
	if !canManageActivity(actor, act) {
		return model.HourRecord{}, model.ErrNotOrganizer
	}
	if h.Status != model.HourPending && h.Status != model.HourDraft {
		return model.HourRecord{}, model.ErrHoursNotPending
	}
	if req.BreakMinutes > 0 {
		h.BreakMinutes = req.BreakMinutes
	}
	h.WorkMinutes = hoursutil.CapMinutes(h.RawMinutes, h.BreakMinutes, act.PlannedMinutes)
	now := s.clock.Now()
	h.Status = model.HourApproved
	h.ReviewerID = actor.ID
	h.ApprovedAt = &now
	h.UpdatedAt = now
	if req.Note != "" {
		h.Note = validate.SanitizePlain(req.Note)
	}
	pts := policy.PointsForMinutes(h.WorkMinutes)
	// 原子入账：把「校验工时仍在待审 + 累加工时与积分 + 写两条流水」合并进
	// store.ApplyHourApproval 的单次写锁内。此前采用先 GetHour 检查状态、后
	// ApplyHoursAndPoints 入账的两步式流程，两次加锁之间存在 TOCTOU 竞态：并发审批
	// 都能读到 pending 而通过校验，随后各自入账，造成工时翻倍、积分翻倍与重复账本。
	// 现在第二个请求在锁内重新读到 approved 即返回 ErrHoursNotPending，保证只入账一次。
	saved, _, err := s.store.ApplyHourApproval(ctx, h, model.HourLedger{
		UserID:     h.VolunteerID,
		HourID:     h.ID,
		ActivityID: h.ActivityID,
		DeltaMin:   h.WorkMinutes,
		Note:       "工时入账",
		CreatedAt:  now,
	}, model.PointLedger{
		UserID:    h.VolunteerID,
		Delta:     pts,
		Reason:    model.PointHours,
		RelatedID: h.ID,
		Note:      "工时积分",
		CreatedAt: now,
	})
	if err != nil {
		return model.HourRecord{}, err
	}
	_ = s.audit.Log(ctx, actor.ID, model.AuditHoursApprove, "hours", saved.ID, act.Title)
	_ = s.notify.Push(ctx, h.VolunteerID, model.NotifyHoursResult, "工时已通过", "「"+act.Title+"」工时 "+formatMinutes(h.WorkMinutes)+" 已入账", h.ID)
	fresh, _ := s.store.GetUserByID(ctx, h.VolunteerID)
	_ = s.cert.MaybeIssue(ctx, actor, fresh)
	return saved, nil
}

func (s *HoursService) Reject(ctx context.Context, actor model.User, hourID string, reason string) (model.HourRecord, error) {
	h, err := s.store.GetHour(ctx, hourID)
	if err != nil {
		return model.HourRecord{}, err
	}
	act, err := s.store.GetActivity(ctx, h.ActivityID)
	if err != nil {
		return model.HourRecord{}, err
	}
	if !canManageActivity(actor, act) {
		return model.HourRecord{}, model.ErrNotOrganizer
	}
	if h.Status != model.HourPending && h.Status != model.HourDraft {
		return model.HourRecord{}, model.ErrHoursNotPending
	}
	h.Status = model.HourRejected
	h.RejectReason = validate.SanitizePlain(reason)
	h.ReviewerID = actor.ID
	h.UpdatedAt = s.clock.Now()
	saved, err := s.store.UpdateHour(ctx, h)
	if err != nil {
		return model.HourRecord{}, err
	}
	_ = s.audit.Log(ctx, actor.ID, model.AuditHoursReject, "hours", saved.ID, saved.RejectReason)
	_ = s.notify.Push(ctx, h.VolunteerID, model.NotifyHoursResult, "工时被驳回", saved.RejectReason, h.ID)
	return saved, nil
}

func (s *HoursService) RequestCorrection(ctx context.Context, actor model.User, hourID string, req model.CorrectHoursRequest) (model.HourRecord, error) {
	h, err := s.store.GetHour(ctx, hourID)
	if err != nil {
		return model.HourRecord{}, err
	}
	if h.VolunteerID != actor.ID {
		return model.HourRecord{}, model.ErrForbidden
	}
	if h.Status != model.HourApproved {
		return model.HourRecord{}, model.ErrHoursNotPending
	}
	if req.RequestedMinutes <= 0 {
		return model.HourRecord{}, model.ErrInvalidMinutes
	}
	h.Status = model.HourPending
	h.CorrectionNote = validate.SanitizePlain(req.Note)
	h.RequestedMins = req.RequestedMinutes
	h.RawMinutes = req.RequestedMinutes
	h.UpdatedAt = s.clock.Now()
	return s.store.UpdateHour(ctx, h)
}

func (s *HoursService) Mine(ctx context.Context, actor model.User) ([]model.HourRecord, error) {
	return s.store.ListHoursByVolunteer(ctx, actor.ID)
}

func (s *HoursService) ByActivity(ctx context.Context, actor model.User, activityID string) ([]model.HourRecord, error) {
	act, err := s.store.GetActivity(ctx, activityID)
	if err != nil {
		return nil, err
	}
	if !canManageActivity(actor, act) {
		return nil, model.ErrNotOrganizer
	}
	return s.store.ListHoursByActivity(ctx, activityID)
}

func (s *HoursService) Pending(ctx context.Context, actor model.User) ([]model.HourRecord, error) {
	if !actor.IsOrganizer() {
		return nil, model.ErrForbidden
	}
	all, err := s.store.ListPendingHours(ctx)
	if err != nil {
		return nil, err
	}
	if actor.IsAdmin() {
		return all, nil
	}
	out := []model.HourRecord{}
	for _, h := range all {
		act, err := s.store.GetActivity(ctx, h.ActivityID)
		if err != nil {
			continue
		}
		if act.OrganizerID == actor.ID {
			out = append(out, h)
		}
	}
	return out, nil
}

func formatMinutes(m int) string {
	h := m / 60
	mm := m % 60
	if h == 0 {
		return itoa(mm) + " 分钟"
	}
	if mm == 0 {
		return itoa(h) + " 小时"
	}
	return itoa(h) + " 小时 " + itoa(mm) + " 分钟"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var digits [12]byte
	i := len(digits)
	for n > 0 {
		i--
		digits[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		digits[i] = '-'
	}
	return string(digits[i:])
}
