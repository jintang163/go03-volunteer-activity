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
