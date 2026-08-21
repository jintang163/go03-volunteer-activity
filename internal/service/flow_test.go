package service

import (
	"context"
	"testing"
	"time"

	"go03-volunteer-activity/internal/auth"
	"go03-volunteer-activity/internal/model"
	"go03-volunteer-activity/internal/store"
)

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
