package store

import (
	"context"
	"time"

	"go03-volunteer-activity/internal/auth"
	"go03-volunteer-activity/internal/model"
	"go03-volunteer-activity/internal/policy"
)

func SeedAdmin(ctx context.Context, st Store, hasher *auth.PasswordHasher, username, password string) error {
	if username == "" {
		username = "admin"
	}
	if password == "" {
		password = "admin123"
	}
	if _, err := st.GetUserByUsername(ctx, username); err == nil {
		return nil
	}
	salt, hash, it, err := hasher.Hash(password)
	if err != nil {
		return err
	}
	now := time.Now()
	_, err = st.CreateUser(ctx, model.User{
		Username:     username,
		DisplayName:  "系统管理员",
		Role:         model.RoleAdmin,
		Status:       model.UserActive,
		PasswordSalt: salt,
		PasswordHash: hash,
		Iterations:   it,
		CreatedAt:    now,
		UpdatedAt:    now,
	})
	return err
}

func SeedDemo(ctx context.Context, st Store, hasher *auth.PasswordHasher) error {
	ensure := func(username, display, password string, role model.UserRole, age int, skills []string) (model.User, error) {
		if u, err := st.GetUserByUsername(ctx, username); err == nil {
			return u, nil
		}
		salt, hash, it, err := hasher.Hash(password)
		if err != nil {
			return model.User{}, err
		}
		now := time.Now()
		return st.CreateUser(ctx, model.User{
			Username:     username,
			DisplayName:  display,
			Role:         role,
			Status:       model.UserActive,
			PasswordSalt: salt,
			PasswordHash: hash,
			Iterations:   it,
			Age:          age,
			Skills:       skills,
			CreatedAt:    now,
			UpdatedAt:    now,
		})
	}
	org, err := ensure("organizer", "社区组织者", "org123", model.RoleOrganizer, 32, nil)
	if err != nil {
		return err
	}
	if _, err := ensure("alice", "爱丽丝", "alice123", model.RoleVolunteer, 22, []string{"环保", "沟通"}); err != nil {
		return err
	}
	if _, err := ensure("bob", "鲍勃", "bob123", model.RoleVolunteer, 28, []string{"搬运"}); err != nil {
		return err
	}

	acts, err := st.ListActivities(ctx, model.ActivityFilter{OrganizerID: org.ID, IncludeDraft: true})
	if err != nil {
		return err
	}
	if len(acts) > 0 {
		return nil
	}
	now := time.Now()
	open := now.Add(-time.Hour)
	closeAt := now.Add(72 * time.Hour)
	start := now.Add(48 * time.Hour)
	end := start.Add(3 * time.Hour)
	code := "DEMO01"
	if ms, ok := st.(*MemoryStore); ok {
		code = ms.NewCheckInCode()
	}
	_, err = st.CreateActivity(ctx, model.Activity{
		OrganizerID:       org.ID,
		Title:             "社区环保清洁",
		Content:           "清理小区公共区域落叶与可回收物，欢迎居民志愿者参加。请穿着便于活动的衣物并自备手套。",
		Category:          model.CatEnvironment,
		Location:          "小区中心花园",
		ContactName:       "社区组织者",
		ContactPhone:      "13800000000",
		Capacity:          20,
		WaitlistEnabled:   true,
		WaitlistLimit:     policy.DefaultWaitlistLimit,
		NeedApproval:      false,
		SignupOpenAt:      open,
		SignupCloseAt:     closeAt,
		StartAt:           start,
		EndAt:             end,
		CheckInOpenBefore: policy.DefaultCheckInOpenBefore,
		CheckOutGrace:     policy.DefaultCheckOutGrace,
		PlannedMinutes:    180,
		MinAge:            16,
		Status:            model.ActivityPublished,
		CheckInCode:       code,
		CreatedAt:         now,
		UpdatedAt:         now,
		PublishedAt:       &now,
	})
	return err
}
