package service

import (
	"context"
	"fmt"
	"time"

	"go03-volunteer-activity/internal/model"
	"go03-volunteer-activity/internal/policy"
	"go03-volunteer-activity/internal/store"
	"go03-volunteer-activity/internal/validate"
)

type ActivityService struct {
	store  store.Store
	notify *NotifyService
	audit  *AuditService
	clock  Clock
}

func NewActivityService(s store.Store, notify *NotifyService, audit *AuditService, clock Clock) *ActivityService {
	return &ActivityService{store: s, notify: notify, audit: audit, clock: clock}
}

func (s *ActivityService) Create(ctx context.Context, actor model.User, req model.CreateActivityRequest) (model.PublicActivity, error) {
	if err := requireActiveWriter(actor); err != nil {
		return model.PublicActivity{}, err
	}
	if !actor.IsOrganizer() {
		return model.PublicActivity{}, model.ErrForbidden
	}
	n, err := s.store.CountActivitiesByOrganizerOpen(ctx, actor.ID)
	if err != nil {
		return model.PublicActivity{}, err
	}
	if n >= policy.MaxOpenActivitiesPerOrg && !actor.IsAdmin() {
		return model.PublicActivity{}, model.ErrConflict
	}
	act, err := s.buildActivity(actor, req)
	if err != nil {
		return model.PublicActivity{}, err
	}
	now := s.clock.Now()
	act.CreatedAt = now
	act.UpdatedAt = now
	if req.Publish {
		if err := s.validatePublish(act); err != nil {
			return model.PublicActivity{}, err
		}
		act.Status = model.ActivityPublished
		act.PublishedAt = &now
	} else {
		act.Status = model.ActivityDraft
	}
	if ms, ok := s.store.(*store.MemoryStore); ok {
		act.CheckInCode = ms.NewCheckInCode()
	}
	saved, err := s.store.CreateActivity(ctx, act)
	if err != nil {
		return model.PublicActivity{}, err
	}
	if saved.Status == model.ActivityPublished {
		_ = s.audit.Log(ctx, actor.ID, model.AuditPublish, "activity", saved.ID, saved.Title)
	}
	return saved.Public(actor), nil
}

func (s *ActivityService) buildActivity(actor model.User, req model.CreateActivityRequest) (model.Activity, error) {
	title := validate.SanitizePlain(req.Title)
	if !validate.InRange(title, 1, policy.TitleMax) {
		return model.Activity{}, model.ErrInvalidTitle
	}
	content := validate.Trim(req.Content)
	if !validate.InRange(content, 1, policy.ContentMax) {
		return model.Activity{}, model.ErrInvalidContent
	}
	if !model.ValidCategory(req.Category) {
		return model.Activity{}, model.ErrInvalidCategory
	}
	if req.Capacity < policy.MinCapacity || req.Capacity > policy.MaxCapacity {
		return model.Activity{}, model.ErrInvalidCapacity
	}
	if req.StartAt.IsZero() || req.EndAt.IsZero() || !req.EndAt.After(req.StartAt) {
		return model.Activity{}, model.ErrInvalidTimeWindow
	}
	if req.SignupOpenAt.IsZero() || req.SignupCloseAt.IsZero() || !req.SignupCloseAt.After(req.SignupOpenAt) {
		return model.Activity{}, model.ErrInvalidTimeWindow
	}
	if req.SignupCloseAt.After(req.EndAt) {
		return model.Activity{}, model.ErrInvalidTimeWindow
	}
	openBefore := req.CheckInOpenBefore
	if openBefore <= 0 {
		openBefore = policy.DefaultCheckInOpenBefore
	}
	grace := req.CheckOutGrace
	if grace <= 0 {
		grace = policy.DefaultCheckOutGrace
	}
	planned := req.PlannedMinutes
	if planned <= 0 {
		planned = int(req.EndAt.Sub(req.StartAt) / time.Minute)
	}
	if planned < policy.MinPlannedMinutes || planned > policy.MaxPlannedMinutes {
		return model.Activity{}, model.ErrInvalidMinutes
	}
	wl := req.WaitlistLimit
	if req.WaitlistEnabled && wl <= 0 {
		wl = policy.DefaultWaitlistLimit
	}
	if req.MinAge < 0 || req.MinAge > 100 {
		return model.Activity{}, model.ErrInvalidAge
	}
	return model.Activity{
		OrganizerID:       actor.ID,
		Title:             title,
		Content:           content,
		Category:          req.Category,
		Location:          validate.SanitizePlain(req.Location),
		ContactName:       validate.SanitizePlain(req.ContactName),
		ContactPhone:      validate.Trim(req.ContactPhone),
		Capacity:          req.Capacity,
		WaitlistEnabled:   req.WaitlistEnabled,
		WaitlistLimit:     wl,
		NeedApproval:      req.NeedApproval,
		SignupOpenAt:      req.SignupOpenAt,
		SignupCloseAt:     req.SignupCloseAt,
		StartAt:           req.StartAt,
		EndAt:             req.EndAt,
		CheckInOpenBefore: openBefore,
		CheckOutGrace:     grace,
		PlannedMinutes:    planned,
		MinAge:            req.MinAge,
		RequiredSkills:    req.RequiredSkills,
		TeamID:            validate.Trim(req.TeamID),
	}, nil
}

func (s *ActivityService) validatePublish(a model.Activity) error {
	if a.Location == "" {
		return fmt.Errorf("%w: location required", model.ErrValidation)
	}
	if a.ContactName == "" {
		return fmt.Errorf("%w: contact required", model.ErrValidation)
	}
	return nil
}

func (s *ActivityService) Update(ctx context.Context, actor model.User, id string, req model.CreateActivityRequest) (model.PublicActivity, error) {
	act, err := s.store.GetActivity(ctx, id)
	if err != nil {
		return model.PublicActivity{}, err
	}
	if !canManageActivity(actor, act) {
		return model.PublicActivity{}, model.ErrNotOrganizer
	}
	if act.Status != model.ActivityDraft && act.Status != model.ActivityPublished {
		return model.PublicActivity{}, model.ErrInvalidActivityStatus
	}
	built, err := s.buildActivity(actor, req)
	if err != nil {
		return model.PublicActivity{}, err
	}
	built.ID = act.ID
	built.OrganizerID = act.OrganizerID
	built.Status = act.Status
	built.CheckInCode = act.CheckInCode
	built.ApprovedCount = act.ApprovedCount
	built.WaitlistCount = act.WaitlistCount
	built.CreatedAt = act.CreatedAt
	built.PublishedAt = act.PublishedAt
	built.UpdatedAt = s.clock.Now()
	saved, err := s.store.UpdateActivity(ctx, built)
	if err != nil {
		return model.PublicActivity{}, err
	}
	return saved.Public(actor), nil
}

func (s *ActivityService) Publish(ctx context.Context, actor model.User, id string) (model.PublicActivity, error) {
	act, err := s.store.GetActivity(ctx, id)
	if err != nil {
		return model.PublicActivity{}, err
	}
	if !canManageActivity(actor, act) {
		return model.PublicActivity{}, model.ErrNotOrganizer
	}
	if act.Status != model.ActivityDraft {
		return model.PublicActivity{}, model.ErrInvalidActivityStatus
	}
	if err := s.validatePublish(act); err != nil {
		return model.PublicActivity{}, err
	}
	now := s.clock.Now()
	act.Status = model.ActivityPublished
	act.PublishedAt = &now
	act.UpdatedAt = now
	saved, err := s.store.UpdateActivity(ctx, act)
	if err != nil {
		return model.PublicActivity{}, err
	}
	_ = s.audit.Log(ctx, actor.ID, model.AuditPublish, "activity", saved.ID, saved.Title)
	return saved.Public(actor), nil
}

func (s *ActivityService) CloseSignup(ctx context.Context, actor model.User, id string) (model.PublicActivity, error) {
	return s.transition(ctx, actor, id, model.ActivityPublished, model.ActivityRegistrationClosed)
}

func (s *ActivityService) Start(ctx context.Context, actor model.User, id string) (model.PublicActivity, error) {
	act, err := s.store.GetActivity(ctx, id)
	if err != nil {
		return model.PublicActivity{}, err
	}
	if !canManageActivity(actor, act) {
		return model.PublicActivity{}, model.ErrNotOrganizer
	}
	if act.Status != model.ActivityPublished && act.Status != model.ActivityRegistrationClosed {
		return model.PublicActivity{}, model.ErrInvalidActivityStatus
	}
	act.Status = model.ActivityInProgress
	act.UpdatedAt = s.clock.Now()
	saved, err := s.store.UpdateActivity(ctx, act)
	if err != nil {
		return model.PublicActivity{}, err
	}
	return saved.Public(actor), nil
}

func (s *ActivityService) Complete(ctx context.Context, actor model.User, id string) (model.PublicActivity, error) {
	act, err := s.store.GetActivity(ctx, id)
	if err != nil {
		return model.PublicActivity{}, err
	}
	if !canManageActivity(actor, act) {
		return model.PublicActivity{}, model.ErrNotOrganizer
	}
	now := s.clock.Now()
	act.Status = model.ActivityCompleted
	act.CompletedAt = &now
	act.UpdatedAt = now
	// 原子地完成「校验活动尚未处于终态 + 写入已完成状态」。此前采用先 GetActivity 读取状态、
	// IsTerminal 检查、后 UpdateActivity 写入的两步式流程，两次加锁之间存在 TOCTOU 竞态：同一场
	// 已结束活动的两个并发完成请求都能读到非终态而通过检查，随后各自写入已完成状态，最终
	// markNoShows 会对同一名未签到志愿者重复标记缺席并扣分两次、发送两次通知。现把状态校验与
	// 写入下沉到 store.ReserveActivityCompletion 的单次写锁内，同一时刻至多一个请求成功迁移状态，
	// 其余在锁内重新读到活动已是 completed 即返回 ErrInvalidActivityStatus，缺席扣分与通知只随
	// 胜出请求触发一次。
	saved, err := s.store.ReserveActivityCompletion(ctx, act)
	if err != nil {
		return model.PublicActivity{}, err
	}
	if err := s.markNoShows(ctx, saved, now); err != nil {
		return model.PublicActivity{}, err
	}
	return saved.Public(actor), nil
}

func (s *ActivityService) markNoShows(ctx context.Context, act model.Activity, now time.Time) error {
	if now.Before(act.EndAt) {
		return nil
	}
	signups, err := s.store.ListSignupsByActivity(ctx, act.ID)
	if err != nil {
		return err
	}
	for _, sg := range signups {
		if sg.Status != model.SignupApproved {
			continue
		}
		_, err := s.store.GetCheckInByActivityVolunteer(ctx, act.ID, sg.VolunteerID)
		if err == nil {
			continue
		}
		sg.Status = model.SignupNoShow
		sg.UpdatedAt = now
		if _, err := s.store.UpdateSignup(ctx, sg); err != nil {
			return err
		}
		u, err := s.store.GetUserByID(ctx, sg.VolunteerID)
		if err != nil {
			continue
		}
		_, _ = s.store.ApplyPoints(ctx, u, model.PointLedger{
			UserID:    u.ID,
			Delta:     -policy.NoShowPointPenalty,
			Reason:    model.PointNoShow,
			RelatedID: act.ID,
			Note:      "活动缺席",
			CreatedAt: now,
		})
		_ = s.notify.Push(ctx, u.ID, model.NotifyNoShow, "缺席记录", "你报名的活动「"+act.Title+"」未签到，已记一次缺席。", act.ID)
	}
	_, _ = s.store.RecountActivityCounters(ctx, act.ID)
	return nil
}

func (s *ActivityService) Cancel(ctx context.Context, actor model.User, id string, reason string) (model.PublicActivity, error) {
	act, err := s.store.GetActivity(ctx, id)
	if err != nil {
		return model.PublicActivity{}, err
	}
	if !canManageActivity(actor, act) {
		return model.PublicActivity{}, model.ErrNotOrganizer
	}
	if act.Status.IsTerminal() {
		return model.PublicActivity{}, model.ErrInvalidActivityStatus
	}
	act.Status = model.ActivityCancelled
	act.CancelReason = validate.SanitizePlain(reason)
	act.UpdatedAt = s.clock.Now()
	saved, err := s.store.UpdateActivity(ctx, act)
	if err != nil {
		return model.PublicActivity{}, err
	}
	_ = s.audit.Log(ctx, actor.ID, model.AuditCancelActivity, "activity", saved.ID, saved.CancelReason)
	signups, _ := s.store.ListSignupsByActivity(ctx, saved.ID)
	for _, sg := range signups {
		if sg.Status.IsActive() {
			_ = s.notify.Push(ctx, sg.VolunteerID, model.NotifyActivity, "活动已取消", "「"+saved.Title+"」已取消。"+saved.CancelReason, saved.ID)
		}
	}
	return saved.Public(actor), nil
}

func (s *ActivityService) RefreshCode(ctx context.Context, actor model.User, id string) (model.PublicActivity, error) {
	act, err := s.store.GetActivity(ctx, id)
	if err != nil {
		return model.PublicActivity{}, err
	}
	if !canManageActivity(actor, act) {
		return model.PublicActivity{}, model.ErrNotOrganizer
	}
	if ms, ok := s.store.(*store.MemoryStore); ok {
		act.CheckInCode = ms.NewCheckInCode()
	} else {
		act.CheckInCode = fmt.Sprintf("%d", s.clock.Now().Unix()%1000000)
	}
	act.UpdatedAt = s.clock.Now()
	saved, err := s.store.UpdateActivity(ctx, act)
	if err != nil {
		return model.PublicActivity{}, err
	}
	return saved.Public(actor), nil
}

func (s *ActivityService) Get(ctx context.Context, actor model.User, id string) (model.PublicActivity, error) {
	act, err := s.maybeAdvance(ctx, id)
	if err != nil {
		return model.PublicActivity{}, err
	}
	if act.Status == model.ActivityDraft && !canManageActivity(actor, act) {
		return model.PublicActivity{}, model.ErrNotFound
	}
	return act.Public(actor), nil
}

func (s *ActivityService) List(ctx context.Context, actor model.User, f model.ActivityFilter) ([]model.PublicActivity, error) {
	if actor.IsOrganizer() && f.OrganizerID == actor.ID {
		f.IncludeDraft = true
	}
	if actor.IsAdmin() {
		f.IncludeDraft = true
	}
	list, err := s.store.ListActivities(ctx, f)
	if err != nil {
		return nil, err
	}
	out := make([]model.PublicActivity, 0, len(list))
	for _, a := range list {
		if a.Status == model.ActivityDraft && !canManageActivity(actor, a) {
			continue
		}
		out = append(out, a.Public(actor))
	}
	return out, nil
}

func (s *ActivityService) maybeAdvance(ctx context.Context, id string) (model.Activity, error) {
	act, err := s.store.GetActivity(ctx, id)
	if err != nil {
		return model.Activity{}, err
	}
	now := s.clock.Now()
	changed := false
	if act.Status == model.ActivityPublished && now.After(act.SignupCloseAt) {
		act.Status = model.ActivityRegistrationClosed
		changed = true
	}
	if (act.Status == model.ActivityPublished || act.Status == model.ActivityRegistrationClosed) &&
		!now.Before(act.StartAt) && now.Before(act.EndAt.Add(time.Duration(act.CheckOutGrace)*time.Minute)) {
		act.Status = model.ActivityInProgress
		changed = true
	}
	if act.Status == model.ActivityInProgress && now.After(act.EndAt.Add(time.Duration(act.CheckOutGrace)*time.Minute)) {
		act.Status = model.ActivityCompleted
		t := now
		act.CompletedAt = &t
		changed = true
	}
	if changed {
		act.UpdatedAt = now
		act, err = s.store.UpdateActivity(ctx, act)
		if err != nil {
			return model.Activity{}, err
		}
		if act.Status == model.ActivityCompleted {
			_ = s.markNoShows(ctx, act, now)
		}
	}
	return act, nil
}

func (s *ActivityService) transition(ctx context.Context, actor model.User, id string, from, to model.ActivityStatus) (model.PublicActivity, error) {
	act, err := s.store.GetActivity(ctx, id)
	if err != nil {
		return model.PublicActivity{}, err
	}
	if !canManageActivity(actor, act) {
		return model.PublicActivity{}, model.ErrNotOrganizer
	}
	if act.Status != from {
		return model.PublicActivity{}, model.ErrInvalidActivityStatus
	}
	act.Status = to
	act.UpdatedAt = s.clock.Now()
	saved, err := s.store.UpdateActivity(ctx, act)
	if err != nil {
		return model.PublicActivity{}, err
	}
	return saved.Public(actor), nil
}

func (s *ActivityService) SubmitFeedback(ctx context.Context, actor model.User, activityID string, req model.FeedbackRequest) (model.Feedback, error) {
	if !actor.IsVolunteer() {
		return model.Feedback{}, model.ErrNotVolunteer
	}
	act, err := s.store.GetActivity(ctx, activityID)
	if err != nil {
		return model.Feedback{}, err
	}
	if act.Status != model.ActivityCompleted {
		return model.Feedback{}, model.ErrActivityNotCompleted
	}
	if req.Score < 1 || req.Score > 5 {
		return model.Feedback{}, model.ErrInvalidScore
	}
	if _, err := s.store.GetCheckInByActivityVolunteer(ctx, activityID, actor.ID); err != nil {
		return model.Feedback{}, model.ErrNotCheckedIn
	}
	if _, err := s.store.GetFeedback(ctx, activityID, actor.ID); err == nil {
		return model.Feedback{}, model.ErrAlreadyFeedback
	}
	// 原子地完成「检查是否已反馈 + 写入」。此前采用先 GetFeedback 检查、后 CreateFeedback
	// 写入的两步式流程，两次加锁之间存在 TOCTOU 竞态：同一志愿者对已完成活动并发提交两次
	// 反馈时，两个请求都能通过存在性检查而各自写入，最终活动保存两条反馈。现把检查与写入
	// 下沉到 store.ReserveFeedback 的单次写锁内，第二个请求在锁内重新读到已存在的反馈记录
	// 即返回 ErrAlreadyFeedback，保证同一志愿者在同一活动只能反馈一次。
	return s.store.ReserveFeedback(ctx, model.Feedback{
		ActivityID:  activityID,
		VolunteerID: actor.ID,
		Score:       req.Score,
		Comment:     validate.SanitizePlain(req.Comment),
		CreatedAt:   s.clock.Now(),
	})
}
