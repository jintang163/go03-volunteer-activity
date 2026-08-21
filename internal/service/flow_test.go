package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"go03-volunteer-activity/internal/auth"
	"go03-volunteer-activity/internal/model"
	"go03-volunteer-activity/internal/store"
)

// capacityBarrierStore 在 ReserveSignup（修复后的原子写入点）上设置同步屏障，
// 让并发报名请求都抵达决策临界区后再同时放行，最大化竞态窗口以稳定复现超额录取。
type capacityBarrierStore struct {
	store.Store
	ready   chan struct{}
	release chan struct{}
}

type hoursApprovalBarrierStore struct {
	store.Store
	ready   chan struct{}
	release chan struct{}
}

type checkInBarrierStore struct {
	store.Store
	ready   chan struct{}
	release chan struct{}
}

// feedbackBarrierStore 在 ReserveFeedback（修复后的原子写入点）上设置同步屏障，
// 让并发反馈请求都抵达决策临界区后再同时放行，最大化竞态窗口以稳定复现重复反馈。
type feedbackBarrierStore struct {
	store.Store
	ready   chan struct{}
	release chan struct{}
}

// teamMemberBarrierStore 在 ReserveTeamMember（修复后的原子写入点）上设置同步屏障，
// 让并发邀请请求都抵达决策临界区后再同时放行，最大化竞态窗口以稳定复现重复成员记录。
type teamMemberBarrierStore struct {
	store.Store
	ready   chan struct{}
	release chan struct{}
}

// checkOutBarrierStore 在 ReserveCheckOut（修复后的原子写入点）上设置同步屏障，
// 让并发签退请求都抵达决策临界区后再同时放行，最大化竞态窗口以稳定复现重复签退。
type checkOutBarrierStore struct {
	store.Store
	ready   chan struct{}
	release chan struct{}
}

// signupApprovalBarrierStore 在 ReserveSignupApproval（修复后的原子写入点）上设置同步屏障，
// 让并发审批请求都抵达决策临界区后再同时放行，最大化竞态窗口以稳定复现超额录取。
type signupApprovalBarrierStore struct {
	store.Store
	ready   chan struct{}
	release chan struct{}
}

// signupCancelBarrierStore 在 ReserveSignupCancellation（修复后的原子写入点）上设置同步屏障，
// 让并发取消请求都抵达决策临界区后再同时放行，最大化竞态窗口以稳定复现重复取消与重复迟到扣分。
type signupCancelBarrierStore struct {
	store.Store
	ready   chan struct{}
	release chan struct{}
}

// noShowBarrierStore 在 ReserveActivityCompletion（修复后的原子写入点）上设置同步屏障，
// 让并发完成请求都抵达决策临界区后再同时放行，最大化竞态窗口以稳定复现重复标记缺席与重复扣分。
type noShowBarrierStore struct {
	store.Store
	ready   chan struct{}
	release chan struct{}
}

// certificateBarrierStore 在 ReserveCertificate（修复后的原子写入点）上设置同步屏障，
// 让并发发证请求都抵达决策临界区后再同时放行，最大化竞态窗口以稳定复现重复发证与重复通知。
type certificateBarrierStore struct {
	store.Store
	ready   chan struct{}
	release chan struct{}
}

func (s *certificateBarrierStore) ReserveCertificate(ctx context.Context, certificate model.Certificate) (model.Certificate, error) {
	s.ready <- struct{}{}
	<-s.release
	return s.Store.ReserveCertificate(ctx, certificate)
}

func (s *noShowBarrierStore) ReserveActivityCompletion(ctx context.Context, activity model.Activity) (model.Activity, error) {
	s.ready <- struct{}{}
	<-s.release
	return s.Store.ReserveActivityCompletion(ctx, activity)
}

func (s *signupCancelBarrierStore) ReserveSignupCancellation(ctx context.Context, signup model.Signup, act model.Activity) (model.Signup, error) {
	s.ready <- struct{}{}
	<-s.release
	return s.Store.ReserveSignupCancellation(ctx, signup, act)
}

func (s *signupApprovalBarrierStore) ReserveSignupApproval(ctx context.Context, signup model.Signup, act model.Activity) (model.Signup, error) {
	s.ready <- struct{}{}
	<-s.release
	return s.Store.ReserveSignupApproval(ctx, signup, act)
}

func (s *checkOutBarrierStore) ReserveCheckOut(ctx context.Context, checkIn model.CheckIn) (model.CheckIn, error) {
	s.ready <- struct{}{}
	<-s.release
	return s.Store.ReserveCheckOut(ctx, checkIn)
}

func (s *teamMemberBarrierStore) ReserveTeamMember(ctx context.Context, member model.TeamMember) (model.TeamMember, error) {
	s.ready <- struct{}{}
	<-s.release
	return s.Store.ReserveTeamMember(ctx, member)
}

func (s *feedbackBarrierStore) ReserveFeedback(ctx context.Context, feedback model.Feedback) (model.Feedback, error) {
	s.ready <- struct{}{}
	<-s.release
	return s.Store.ReserveFeedback(ctx, feedback)
}

// checkInBarrierStore 在 ReserveCheckIn（修复后的原子写入点）上设置同步屏障，
// 让并发签到请求都抵达决策临界区后再同时放行，最大化竞态窗口以稳定复现重复签到。
func (s *checkInBarrierStore) ReserveCheckIn(ctx context.Context, checkIn model.CheckIn) (model.CheckIn, error) {
	s.ready <- struct{}{}
	<-s.release
	return s.Store.ReserveCheckIn(ctx, checkIn)
}

func (s *hoursApprovalBarrierStore) ApplyHourApproval(ctx context.Context, hour model.HourRecord, hl model.HourLedger, pl model.PointLedger) (model.HourRecord, model.User, error) {
	s.ready <- struct{}{}
	<-s.release
	return s.Store.ApplyHourApproval(ctx, hour, hl, pl)
}

func (s *capacityBarrierStore) ReserveSignup(ctx context.Context, sg model.Signup, act model.Activity) (model.Signup, error) {
	s.ready <- struct{}{}
	<-s.release
	return s.Store.ReserveSignup(ctx, sg, act)
}

type fakeClock struct{ t time.Time }

func (f fakeClock) Now() time.Time { return f.t }

func testEnv(t *testing.T, now time.Time) (*Services, store.Store, *auth.PasswordHasher) {
	t.Helper()
	st := store.NewMemoryStore(nil, nil)
	h := auth.NewPasswordHasher()
	sm := auth.NewSessionManager(time.Hour)
	svc := NewServices(st, h, sm, fakeClock{t: now}, 3)
	return svc, st, h
}

func mustUser(t *testing.T, st store.Store, h *auth.PasswordHasher, name, pass string, role model.UserRole) model.User {
	t.Helper()
	salt, hash, it, err := h.Hash(pass)
	if err != nil {
		t.Fatal(err)
	}
	u, err := st.CreateUser(context.Background(), model.User{
		Username: name, DisplayName: name, Role: role, Status: model.UserActive,
		PasswordSalt: salt, PasswordHash: hash, Iterations: it, Age: 20,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func TestSignupWaitlistAndApprove(t *testing.T) {
	now := time.Date(2026, 8, 21, 8, 0, 0, 0, time.UTC)
	svc, st, hasher := testEnv(t, now)
	ctx := context.Background()
	org := mustUser(t, st, hasher, "org", "org123x", model.RoleOrganizer)
	v1 := mustUser(t, st, hasher, "v1", "pass123", model.RoleVolunteer)
	v2 := mustUser(t, st, hasher, "v2", "pass123", model.RoleVolunteer)
	act, err := svc.Activity.Create(ctx, org, model.CreateActivityRequest{
		Title: "环保清洁", Content: "清理花园落叶与可回收物活动说明", Category: model.CatEnvironment,
		Location: "花园", ContactName: "org", Capacity: 1, WaitlistEnabled: true, WaitlistLimit: 10,
		SignupOpenAt: now.Add(-time.Hour), SignupCloseAt: now.Add(24 * time.Hour),
		StartAt: now.Add(48 * time.Hour), EndAt: now.Add(51 * time.Hour),
		PlannedMinutes: 180, Publish: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	s1, err := svc.Signup.Signup(ctx, v1, act.ID, model.SignupRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if s1.Status != model.SignupApproved {
		t.Fatalf("s1=%s", s1.Status)
	}
	s2, err := svc.Signup.Signup(ctx, v2, act.ID, model.SignupRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if s2.Status != model.SignupWaitlisted {
		t.Fatalf("s2=%s", s2.Status)
	}
	if _, err := svc.Signup.Cancel(ctx, v1, s1.ID); err != nil {
		t.Fatal(err)
	}
	got, err := st.GetSignup(ctx, s2.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != model.SignupApproved {
		t.Fatalf("promoted=%s", got.Status)
	}
}

func TestConcurrentSignupRespectsCapacity(t *testing.T) {
	now := time.Date(2026, 8, 21, 8, 0, 0, 0, time.UTC)
	base := store.NewMemoryStore(nil, nil)
	barrier := &capacityBarrierStore{
		Store:   base,
		ready:   make(chan struct{}, 2),
		release: make(chan struct{}),
	}
	hasher := auth.NewPasswordHasher()
	svc := NewServices(barrier, hasher, auth.NewSessionManager(time.Hour), fakeClock{t: now}, 3)
	ctx := context.Background()
	org := mustUser(t, base, hasher, "capacity-org", "org123x", model.RoleOrganizer)
	v1 := mustUser(t, base, hasher, "capacity-v1", "pass123", model.RoleVolunteer)
	v2 := mustUser(t, base, hasher, "capacity-v2", "pass123", model.RoleVolunteer)
	act, err := svc.Activity.Create(ctx, org, model.CreateActivityRequest{
		Title: "并发报名活动", Content: "验证活动满员时不会发生超额录取的业务场景", Category: model.CatCommunity,
		Location: "社区中心", ContactName: "org", Capacity: 1, WaitlistEnabled: false,
		SignupOpenAt: now.Add(-time.Hour), SignupCloseAt: now.Add(time.Hour),
		StartAt: now.Add(2 * time.Hour), EndAt: now.Add(4 * time.Hour), PlannedMinutes: 120, Publish: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	type result struct {
		signup model.Signup
		err    error
	}
	results := make(chan result, 2)
	var wg sync.WaitGroup
	for _, volunteer := range []model.User{v1, v2} {
		wg.Add(1)
		go func(actor model.User) {
			defer wg.Done()
			signup, signupErr := svc.Signup.Signup(ctx, actor, act.ID, model.SignupRequest{})
			results <- result{signup: signup, err: signupErr}
		}(volunteer)
	}

	<-barrier.ready
	<-barrier.ready
	close(barrier.release)
	wg.Wait()
	close(results)

	approved, full := 0, 0
	for result := range results {
		switch {
		case result.err == nil && result.signup.Status == model.SignupApproved:
			approved++
		case errors.Is(result.err, model.ErrCapacityFull):
			full++
		default:
			t.Fatalf("unexpected signup result: signup=%+v err=%v", result.signup, result.err)
		}
	}
	if approved != 1 || full != 1 {
		t.Fatalf("capacity=1 concurrent results: approved=%d capacity_full=%d; want approved=1 capacity_full=1", approved, full)
	}
	stored, err := base.ListSignupsByActivity(ctx, act.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(stored) != 1 || stored[0].Status != model.SignupApproved {
		t.Fatalf("stored signups=%+v; want exactly one approved signup", stored)
	}
}

func TestCheckInRequiresCodeAndApproval(t *testing.T) {
	now := time.Date(2026, 8, 21, 10, 0, 0, 0, time.UTC)
	svc, st, hasher := testEnv(t, now)
	ctx := context.Background()
	org := mustUser(t, st, hasher, "org2", "org123x", model.RoleOrganizer)
	vol := mustUser(t, st, hasher, "vol", "pass123", model.RoleVolunteer)
	act, err := svc.Activity.Create(ctx, org, model.CreateActivityRequest{
		Title: "助老探访", Content: "探望社区独居老人并协助采购生活用品", Category: model.CatElderly,
		Location: "2栋", ContactName: "org", Capacity: 5,
		SignupOpenAt: now.Add(-2 * time.Hour), SignupCloseAt: now.Add(time.Hour),
		StartAt: now, EndAt: now.Add(2 * time.Hour), PlannedMinutes: 120, Publish: true,
		CheckInOpenBefore: 30, CheckOutGrace: 60,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CheckIn.SelfCheckIn(ctx, vol, act.ID, model.CheckInRequest{Code: act.CheckInCode}); err == nil {
		t.Fatal("expected not approved")
	}
	if _, err := svc.Signup.Signup(ctx, vol, act.ID, model.SignupRequest{}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CheckIn.SelfCheckIn(ctx, vol, act.ID, model.CheckInRequest{Code: "WRONG"}); err == nil {
		t.Fatal("expected wrong code")
	}
	ci, err := svc.CheckIn.SelfCheckIn(ctx, vol, act.ID, model.CheckInRequest{Code: act.CheckInCode})
	if err != nil {
		t.Fatal(err)
	}
	later := now.Add(90 * time.Minute)
	svc.CheckIn.clock = fakeClock{t: later}
	co, err := svc.CheckIn.CheckOut(ctx, vol, act.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !co.HasCheckedOut() {
		t.Fatal("checkout missing")
	}
	hrs, err := st.ListHoursByVolunteer(ctx, vol.ID)
	if err != nil || len(hrs) != 1 {
		t.Fatalf("hours=%v %v", hrs, err)
	}
	if hrs[0].WorkMinutes != 90 {
		t.Fatalf("work=%d ci=%v", hrs[0].WorkMinutes, ci.CheckInAt)
	}
}

func TestHoursApproveAddsPoints(t *testing.T) {
	now := time.Date(2026, 8, 21, 10, 0, 0, 0, time.UTC)
	svc, st, hasher := testEnv(t, now)
	ctx := context.Background()
	org := mustUser(t, st, hasher, "org3", "org123x", model.RoleOrganizer)
	vol := mustUser(t, st, hasher, "vol3", "pass123", model.RoleVolunteer)
	act, err := svc.Activity.Create(ctx, org, model.CreateActivityRequest{
		Title: "社区值守", Content: "协助社区入口值守与秩序维护工作说明", Category: model.CatCommunity,
		Location: "门口", ContactName: "org", Capacity: 5,
		SignupOpenAt: now.Add(-time.Hour), SignupCloseAt: now.Add(time.Hour),
		StartAt: now, EndAt: now.Add(3 * time.Hour), PlannedMinutes: 180, Publish: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Signup.Signup(ctx, vol, act.ID, model.SignupRequest{}); err != nil {
		t.Fatal(err)
	}
	hrec, err := svc.Hours.SubmitManual(ctx, org, model.SubmitHoursRequest{
		ActivityID: act.ID, VolunteerID: vol.ID, WorkMinutes: 60,
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := svc.Hours.Approve(ctx, org, hrec.ID, model.ReviewHoursRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != model.HourApproved {
		t.Fatalf("status=%s", got.Status)
	}
	u, _ := st.GetUserByID(ctx, vol.ID)
	if u.TotalMinutes != 60 {
		t.Fatalf("minutes=%d", u.TotalMinutes)
	}
	if u.Points != 10 {
		t.Fatalf("points=%d", u.Points)
	}
}

func TestConcurrentHoursApprovalIsIdempotent(t *testing.T) {
	now := time.Date(2026, 8, 21, 10, 0, 0, 0, time.UTC)
	base := store.NewMemoryStore(nil, nil)
	barrier := &hoursApprovalBarrierStore{
		Store:   base,
		ready:   make(chan struct{}, 2),
		release: make(chan struct{}),
	}
	hasher := auth.NewPasswordHasher()
	svc := NewServices(barrier, hasher, auth.NewSessionManager(time.Hour), fakeClock{t: now}, 3)
	ctx := context.Background()
	org := mustUser(t, base, hasher, "hours-org", "org123x", model.RoleOrganizer)
	vol := mustUser(t, base, hasher, "hours-vol", "pass123", model.RoleVolunteer)
	act, err := svc.Activity.Create(ctx, org, model.CreateActivityRequest{
		Title: "并发工时审批活动", Content: "验证同一工时记录不会被重复审批和重复入账", Category: model.CatCommunity,
		Location: "服务站", ContactName: "org", Capacity: 5,
		SignupOpenAt: now.Add(-time.Hour), SignupCloseAt: now.Add(time.Hour),
		StartAt: now, EndAt: now.Add(2 * time.Hour), PlannedMinutes: 120, Publish: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Signup.Signup(ctx, vol, act.ID, model.SignupRequest{}); err != nil {
		t.Fatal(err)
	}
	hour, err := svc.Hours.SubmitManual(ctx, org, model.SubmitHoursRequest{
		ActivityID: act.ID, VolunteerID: vol.ID, WorkMinutes: 60,
	})
	if err != nil {
		t.Fatal(err)
	}

	type result struct {
		hour model.HourRecord
		err  error
	}
	results := make(chan result, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			approved, approveErr := svc.Hours.Approve(ctx, org, hour.ID, model.ReviewHoursRequest{})
			results <- result{hour: approved, err: approveErr}
		}()
	}
	<-barrier.ready
	<-barrier.ready
	close(barrier.release)
	wg.Wait()
	close(results)

	approved, notPending := 0, 0
	for result := range results {
		switch {
		case result.err == nil && result.hour.Status == model.HourApproved:
			approved++
		case errors.Is(result.err, model.ErrHoursNotPending):
			notPending++
		default:
			t.Fatalf("unexpected approval result: hour=%+v err=%v", result.hour, result.err)
		}
	}
	user, err := base.GetUserByID(ctx, vol.ID)
	if err != nil {
		t.Fatal(err)
	}
	hourLedgers, err := base.ListHourLedgers(ctx, vol.ID)
	if err != nil {
		t.Fatal(err)
	}
	pointLedgers, err := base.ListPointLedgers(ctx, vol.ID)
	if err != nil {
		t.Fatal(err)
	}
	if approved != 1 || notPending != 1 || user.TotalMinutes != 60 || user.Points != 10 || len(hourLedgers) != 1 || len(pointLedgers) != 1 {
		t.Fatalf("concurrent approval result: approved=%d not_pending=%d minutes=%d points=%d hour_ledgers=%d point_ledgers=%d; want 1,1,60,10,1,1", approved, notPending, user.TotalMinutes, user.Points, len(hourLedgers), len(pointLedgers))
	}
}

func TestConcurrentSelfCheckInCreatesSingleRecord(t *testing.T) {
	now := time.Date(2026, 8, 21, 10, 0, 0, 0, time.UTC)
	base := store.NewMemoryStore(nil, nil)
	barrier := &checkInBarrierStore{Store: base, ready: make(chan struct{}, 2), release: make(chan struct{})}
	hasher := auth.NewPasswordHasher()
	svc := NewServices(barrier, hasher, auth.NewSessionManager(time.Hour), fakeClock{t: now}, 3)
	ctx := context.Background()
	org := mustUser(t, base, hasher, "checkin-org", "org123x", model.RoleOrganizer)
	vol := mustUser(t, base, hasher, "checkin-vol", "pass123", model.RoleVolunteer)
	act, err := svc.Activity.Create(ctx, org, model.CreateActivityRequest{
		Title: "并发签到活动", Content: "验证同一志愿者不会在同一活动产生重复签到记录", Category: model.CatCommunity,
		Location: "签到点", ContactName: "org", Capacity: 5,
		SignupOpenAt: now.Add(-2 * time.Hour), SignupCloseAt: now.Add(time.Hour),
		StartAt: now, EndAt: now.Add(2 * time.Hour), PlannedMinutes: 120, Publish: true,
		CheckInOpenBefore: 30, CheckOutGrace: 60,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Signup.Signup(ctx, vol, act.ID, model.SignupRequest{}); err != nil {
		t.Fatal(err)
	}

	type result struct {
		checkIn model.CheckIn
		err     error
	}
	results := make(chan result, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			record, checkInErr := svc.CheckIn.SelfCheckIn(ctx, vol, act.ID, model.CheckInRequest{Code: act.CheckInCode})
			results <- result{checkIn: record, err: checkInErr}
		}()
	}
	<-barrier.ready
	<-barrier.ready
	close(barrier.release)
	wg.Wait()
	close(results)

	succeeded, duplicate := 0, 0
	for result := range results {
		switch {
		case result.err == nil && result.checkIn.ID != "":
			succeeded++
		case errors.Is(result.err, model.ErrAlreadyCheckedIn):
			duplicate++
		default:
			t.Fatalf("unexpected check-in result: record=%+v err=%v", result.checkIn, result.err)
		}
	}
	stored, err := base.ListCheckInsByActivity(ctx, act.ID)
	if err != nil {
		t.Fatal(err)
	}
	if succeeded != 1 || duplicate != 1 || len(stored) != 1 {
		t.Fatalf("concurrent check-in result: succeeded=%d already_checked_in=%d stored=%d; want 1,1,1", succeeded, duplicate, len(stored))
	}
}

func TestConcurrentFeedbackCreatesSingleSubmission(t *testing.T) {
	now := time.Date(2026, 8, 21, 15, 0, 0, 0, time.UTC)
	base := store.NewMemoryStore(nil, nil)
	barrier := &feedbackBarrierStore{Store: base, ready: make(chan struct{}, 2), release: make(chan struct{})}
	hasher := auth.NewPasswordHasher()
	svc := NewServices(barrier, hasher, auth.NewSessionManager(time.Hour), fakeClock{t: now}, 3)
	ctx := context.Background()
	org := mustUser(t, base, hasher, "feedback-org", "org123x", model.RoleOrganizer)
	vol := mustUser(t, base, hasher, "feedback-vol", "pass123", model.RoleVolunteer)
	act, err := base.CreateActivity(ctx, model.Activity{
		OrganizerID: org.ID, Title: "已完成反馈活动", Content: "验证同一志愿者不会重复提交反馈",
		Category: model.CatCommunity, Location: "社区", ContactName: "org", Capacity: 5,
		StartAt: now.Add(-3 * time.Hour), EndAt: now.Add(-time.Hour), PlannedMinutes: 120,
		Status: model.ActivityCompleted, CreatedAt: now.Add(-24 * time.Hour), UpdatedAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := base.CreateCheckIn(ctx, model.CheckIn{ActivityID: act.ID, VolunteerID: vol.ID, CheckInAt: act.StartAt, CreatedAt: act.StartAt}); err != nil {
		t.Fatal(err)
	}

	type result struct {
		feedback model.Feedback
		err      error
	}
	results := make(chan result, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			feedback, feedbackErr := svc.Activity.SubmitFeedback(ctx, vol, act.ID, model.FeedbackRequest{Score: 5, Comment: "活动很好"})
			results <- result{feedback: feedback, err: feedbackErr}
		}()
	}
	<-barrier.ready
	<-barrier.ready
	close(barrier.release)
	wg.Wait()
	close(results)

	succeeded, duplicate := 0, 0
	for result := range results {
		switch {
		case result.err == nil && result.feedback.ID != "":
			succeeded++
		case errors.Is(result.err, model.ErrAlreadyFeedback):
			duplicate++
		default:
			t.Fatalf("unexpected feedback result: feedback=%+v err=%v", result.feedback, result.err)
		}
	}
	stored, err := base.ListFeedbackByActivity(ctx, act.ID)
	if err != nil {
		t.Fatal(err)
	}
	if succeeded != 1 || duplicate != 1 || len(stored) != 1 {
		t.Fatalf("concurrent feedback result: succeeded=%d already_feedback=%d stored=%d; want 1,1,1", succeeded, duplicate, len(stored))
	}
}

func TestConcurrentTeamInviteCreatesSingleMembership(t *testing.T) {
	now := time.Date(2026, 8, 21, 15, 0, 0, 0, time.UTC)
	base := store.NewMemoryStore(nil, nil)
	hasher := auth.NewPasswordHasher()
	ctx := context.Background()
	owner := mustUser(t, base, hasher, "team-owner", "org123x", model.RoleOrganizer)
	member := mustUser(t, base, hasher, "team-member", "pass123", model.RoleVolunteer)
	team, err := base.CreateTeam(ctx, model.Team{OwnerID: owner.ID, Name: "并发邀请团队", CreatedAt: now, UpdatedAt: now})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := base.CreateTeamMember(ctx, model.TeamMember{TeamID: team.ID, UserID: owner.ID, Role: "owner", JoinedAt: now}); err != nil {
		t.Fatal(err)
	}
	barrier := &teamMemberBarrierStore{Store: base, ready: make(chan struct{}, 2), release: make(chan struct{})}
	svc := NewServices(barrier, hasher, auth.NewSessionManager(time.Hour), fakeClock{t: now}, 3)

	type result struct {
		member model.TeamMember
		err    error
	}
	results := make(chan result, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			created, inviteErr := svc.Team.Invite(ctx, owner, team.ID, member.Username)
			results <- result{member: created, err: inviteErr}
		}()
	}
	<-barrier.ready
	<-barrier.ready
	close(barrier.release)
	wg.Wait()
	close(results)

	succeeded, duplicate := 0, 0
	for result := range results {
		switch {
		case result.err == nil && result.member.ID != "":
			succeeded++
		case errors.Is(result.err, model.ErrAlreadyTeamMember):
			duplicate++
		default:
			t.Fatalf("unexpected invite result: member=%+v err=%v", result.member, result.err)
		}
	}
	members, err := base.ListTeamMembers(ctx, team.ID)
	if err != nil {
		t.Fatal(err)
	}
	targetCount := 0
	for _, stored := range members {
		if stored.UserID == member.ID {
			targetCount++
		}
	}
	if succeeded != 1 || duplicate != 1 || targetCount != 1 {
		t.Fatalf("concurrent invite result: succeeded=%d already_member=%d target_memberships=%d; want 1,1,1", succeeded, duplicate, targetCount)
	}
}

func TestConcurrentCheckOutTransitionsOnce(t *testing.T) {
	start := time.Date(2026, 8, 21, 10, 0, 0, 0, time.UTC)
	now := start.Add(90 * time.Minute)
	base := store.NewMemoryStore(nil, nil)
	hasher := auth.NewPasswordHasher()
	ctx := context.Background()
	org := mustUser(t, base, hasher, "checkout-org", "org123x", model.RoleOrganizer)
	vol := mustUser(t, base, hasher, "checkout-vol", "pass123", model.RoleVolunteer)
	act, err := base.CreateActivity(ctx, model.Activity{OrganizerID: org.ID, Title: "并发签退活动", Capacity: 5, StartAt: start, EndAt: start.Add(2 * time.Hour), PlannedMinutes: 120, CheckOutGrace: 60, Status: model.ActivityInProgress, CreatedAt: start})
	if err != nil {
		t.Fatal(err)
	}
	signup, err := base.CreateSignup(ctx, model.Signup{ActivityID: act.ID, VolunteerID: vol.ID, Status: model.SignupApproved, CreatedAt: start.Add(-time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := base.CreateCheckIn(ctx, model.CheckIn{ActivityID: act.ID, VolunteerID: vol.ID, SignupID: signup.ID, CheckInAt: start, CreatedAt: start}); err != nil {
		t.Fatal(err)
	}
	barrier := &checkOutBarrierStore{Store: base, ready: make(chan struct{}, 2), release: make(chan struct{})}
	svc := NewServices(barrier, hasher, auth.NewSessionManager(time.Hour), fakeClock{t: now}, 3)

	type result struct {
		checkIn model.CheckIn
		err     error
	}
	results := make(chan result, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			updated, checkOutErr := svc.CheckIn.CheckOut(ctx, vol, act.ID)
			results <- result{checkIn: updated, err: checkOutErr}
		}()
	}
	<-barrier.ready
	<-barrier.ready
	close(barrier.release)
	wg.Wait()
	close(results)

	succeeded, already := 0, 0
	for result := range results {
		switch {
		case result.err == nil && result.checkIn.HasCheckedOut():
			succeeded++
		case errors.Is(result.err, model.ErrAlreadyCheckedOut):
			already++
		default:
			t.Fatalf("unexpected check-out result: record=%+v err=%v", result.checkIn, result.err)
		}
	}
	if succeeded != 1 || already != 1 {
		t.Fatalf("concurrent check-out result: succeeded=%d already_checked_out=%d; want 1,1", succeeded, already)
	}
	// 签退状态只能迁移一次意味着 autoDraftHours 也只能随胜出请求触发一次：
	// 同一志愿者在该活动下只能存在一条工时草拟，且没有任何工时/积分流水被重复生成。
	hours, err := base.ListHoursByActivity(ctx, act.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(hours) != 1 {
		t.Fatalf("concurrent check-out hours=%d; want 1 (auto-draft must not fire twice)", len(hours))
	}
	if hours[0].WorkMinutes != 90 {
		t.Fatalf("auto-draft work minutes=%d; want 90", hours[0].WorkMinutes)
	}
	hourLedgers, err := base.ListHourLedgers(ctx, vol.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(hourLedgers) != 0 {
		t.Fatalf("concurrent check-out hour_ledgers=%d; want 0 (draft does not post ledger)", len(hourLedgers))
	}
}

func TestConcurrentSignupApprovalRespectsLastCapacitySlot(t *testing.T) {
	now := time.Date(2026, 8, 21, 8, 0, 0, 0, time.UTC)
	base := store.NewMemoryStore(nil, nil)
	hasher := auth.NewPasswordHasher()
	setup := NewServices(base, hasher, auth.NewSessionManager(time.Hour), fakeClock{t: now}, 3)
	ctx := context.Background()
	org := mustUser(t, base, hasher, "approval-org", "org123x", model.RoleOrganizer)
	v1 := mustUser(t, base, hasher, "approval-v1", "pass123", model.RoleVolunteer)
	v2 := mustUser(t, base, hasher, "approval-v2", "pass123", model.RoleVolunteer)
	act, err := setup.Activity.Create(ctx, org, model.CreateActivityRequest{
		Title: "并发审批活动", Content: "用于验证最后一个活动名额不会被并发审批重复占用", Category: model.CatCommunity,
		Location: "社区", ContactName: "org", Capacity: 1, NeedApproval: true,
		SignupOpenAt: now.Add(-time.Hour), SignupCloseAt: now.Add(24 * time.Hour),
		StartAt: now.Add(48 * time.Hour), EndAt: now.Add(50 * time.Hour), PlannedMinutes: 120, Publish: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	s1, err := setup.Signup.Signup(ctx, v1, act.ID, model.SignupRequest{})
	if err != nil {
		t.Fatal(err)
	}
	s2, err := setup.Signup.Signup(ctx, v2, act.ID, model.SignupRequest{})
	if err != nil {
		t.Fatal(err)
	}
	barrier := &signupApprovalBarrierStore{Store: base, ready: make(chan struct{}, 2), release: make(chan struct{})}
	svc := NewServices(barrier, hasher, auth.NewSessionManager(time.Hour), fakeClock{t: now}, 3)
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for _, id := range []string{s1.ID, s2.ID} {
		wg.Add(1)
		go func(signupID string) {
			defer wg.Done()
			_, err := svc.Signup.Approve(ctx, org, signupID)
			errs <- err
		}(id)
	}
	<-barrier.ready
	<-barrier.ready
	close(barrier.release)
	wg.Wait()
	close(errs)
	succeeded, full := 0, 0
	for err := range errs {
		switch {
		case err == nil:
			succeeded++
		case errors.Is(err, model.ErrCapacityFull):
			full++
		default:
			t.Fatalf("unexpected approval error: %v", err)
		}
	}
	approved, err := base.CountApprovedByActivity(ctx, act.ID)
	if err != nil {
		t.Fatal(err)
	}
	if succeeded != 1 || full != 1 || approved != 1 {
		t.Fatalf("concurrent approvals: succeeded=%d capacity_full=%d approved=%d; want 1,1,1", succeeded, full, approved)
	}
}

func TestConcurrentLateCancellationAppliesPenaltyOnce(t *testing.T) {
	open := time.Date(2026, 8, 21, 8, 0, 0, 0, time.UTC)
	base := store.NewMemoryStore(nil, nil)
	hasher := auth.NewPasswordHasher()
	setup := NewServices(base, hasher, auth.NewSessionManager(time.Hour), fakeClock{t: open}, 3)
	ctx := context.Background()
	org := mustUser(t, base, hasher, "cancel-org", "org123x", model.RoleOrganizer)
	vol := mustUser(t, base, hasher, "cancel-vol", "pass123", model.RoleVolunteer)
	act, err := setup.Activity.Create(ctx, org, model.CreateActivityRequest{Title: "迟到取消活动", Content: "验证并发取消只产生一次迟到扣分", Category: model.CatCommunity, Location: "社区", ContactName: "org", Capacity: 2, SignupOpenAt: open.Add(-time.Hour), SignupCloseAt: open.Add(time.Hour), StartAt: open.Add(2 * time.Hour), EndAt: open.Add(4 * time.Hour), PlannedMinutes: 120, Publish: true})
	if err != nil {
		t.Fatal(err)
	}
	sg, err := setup.Signup.Signup(ctx, vol, act.ID, model.SignupRequest{})
	if err != nil {
		t.Fatal(err)
	}
	barrier := &signupCancelBarrierStore{Store: base, ready: make(chan struct{}, 2), release: make(chan struct{})}
	svc := NewServices(barrier, hasher, auth.NewSessionManager(time.Hour), fakeClock{t: open.Add(3 * time.Hour)}, 3)
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, err := svc.Signup.Cancel(ctx, vol, sg.ID); errs <- err }()
	}
	<-barrier.ready
	<-barrier.ready
	close(barrier.release)
	wg.Wait()
	close(errs)
	succeeded, conflicts := 0, 0
	for err := range errs {
		if err == nil {
			succeeded++
		} else if errors.Is(err, model.ErrConflict) {
			conflicts++
		} else {
			t.Fatalf("unexpected cancel error: %v", err)
		}
	}
	ledgers, err := base.ListPointLedgers(ctx, vol.ID)
	if err != nil {
		t.Fatal(err)
	}
	if succeeded != 1 || conflicts != 1 || len(ledgers) != 1 {
		t.Fatalf("concurrent late cancellation: succeeded=%d conflicts=%d point_ledgers=%d; want 1,1,1", succeeded, conflicts, len(ledgers))
	}
}

func TestConcurrentActivityCompletionMarksNoShowOnce(t *testing.T) {
	open := time.Date(2026, 8, 21, 8, 0, 0, 0, time.UTC)
	base := store.NewMemoryStore(nil, nil)
	hasher := auth.NewPasswordHasher()
	setup := NewServices(base, hasher, auth.NewSessionManager(time.Hour), fakeClock{t: open}, 3)
	ctx := context.Background()
	org := mustUser(t, base, hasher, "complete-org", "org123x", model.RoleOrganizer)
	vol := mustUser(t, base, hasher, "complete-vol", "pass123", model.RoleVolunteer)
	act, err := setup.Activity.Create(ctx, org, model.CreateActivityRequest{Title: "并发完成活动", Content: "验证并发完成不会重复记录缺席扣分", Category: model.CatCommunity, Location: "社区", ContactName: "org", Capacity: 2, SignupOpenAt: open.Add(-time.Hour), SignupCloseAt: open.Add(time.Hour), StartAt: open.Add(2 * time.Hour), EndAt: open.Add(4 * time.Hour), PlannedMinutes: 120, Publish: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := setup.Signup.Signup(ctx, vol, act.ID, model.SignupRequest{}); err != nil {
		t.Fatal(err)
	}
	barrier := &noShowBarrierStore{Store: base, ready: make(chan struct{}, 2), release: make(chan struct{})}
	svc := NewServices(barrier, hasher, auth.NewSessionManager(time.Hour), fakeClock{t: open.Add(5 * time.Hour)}, 3)
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, err := svc.Activity.Complete(ctx, org, act.ID); errs <- err }()
	}
	<-barrier.ready
	<-barrier.ready
	close(barrier.release)
	wg.Wait()
	close(errs)
	succeeded, invalid := 0, 0
	for err := range errs {
		if err == nil {
			succeeded++
		} else if errors.Is(err, model.ErrInvalidActivityStatus) {
			invalid++
		} else {
			t.Fatalf("unexpected complete error: %v", err)
		}
	}
	ledgers, err := base.ListPointLedgers(ctx, vol.ID)
	if err != nil {
		t.Fatal(err)
	}
	if succeeded != 1 || invalid != 1 || len(ledgers) != 1 {
		t.Fatalf("concurrent completion: succeeded=%d invalid_status=%d point_ledgers=%d; want 1,1,1", succeeded, invalid, len(ledgers))
	}
}

func TestConcurrentCertificateIssueCreatesSingleTier(t *testing.T) {
	now := time.Date(2026, 8, 21, 8, 0, 0, 0, time.UTC)
	base := store.NewMemoryStore(nil, nil)
	hasher := auth.NewPasswordHasher()
	actor := mustUser(t, base, hasher, "cert-admin", "admin123", model.RoleAdmin)
	vol := mustUser(t, base, hasher, "cert-vol", "pass123", model.RoleVolunteer)
	vol.TotalMinutes = 20 * 60
	vol, err := base.UpdateUser(context.Background(), vol)
	if err != nil {
		t.Fatal(err)
	}
	barrier := &certificateBarrierStore{Store: base, ready: make(chan struct{}, 2), release: make(chan struct{})}
	svc := NewServices(barrier, hasher, auth.NewSessionManager(time.Hour), fakeClock{t: now}, 3)
	ctx := context.Background()
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); errs <- svc.Cert.MaybeIssue(ctx, actor, vol) }()
	}
	<-barrier.ready
	<-barrier.ready
	close(barrier.release)
	wg.Wait()
	close(errs)
	succeeded, duplicate := 0, 0
	for err := range errs {
		switch {
		case err == nil:
			succeeded++
		case errors.Is(err, model.ErrAlreadyCertTier):
			duplicate++
		default:
			t.Fatalf("unexpected certificate issue error: %v", err)
		}
	}
	certs, err := base.ListCertificatesByUser(ctx, vol.ID)
	if err != nil {
		t.Fatal(err)
	}
	notifications, err := base.ListNotifications(ctx, vol.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	audits, err := base.ListAudits(ctx, "certificate", "")
	if err != nil {
		t.Fatal(err)
	}
	// 只有一个请求胜出发证，第二个在锁内重新读到同档位证书即返回 ErrAlreadyCertTier，
	// 因此证书、通知与审计都只随胜出请求各产生一份，重复请求不再产生额外副作用。
	if succeeded != 1 || duplicate != 1 || len(certs) != 1 || len(notifications) != 1 || len(audits) != 1 {
		t.Fatalf("concurrent certificate issue: succeeded=%d already_cert=%d certificates=%d notifications=%d audits=%d; want 1,1,1,1,1", succeeded, duplicate, len(certs), len(notifications), len(audits))
	}
}
