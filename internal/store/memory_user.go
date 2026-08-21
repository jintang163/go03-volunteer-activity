package store

import (
	"context"
	"strings"
	"time"

	"go03-volunteer-activity/internal/model"
)

func (m *MemoryStore) CreateUser(_ context.Context, u model.User) (model.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	name := strings.ToLower(strings.TrimSpace(u.Username))
	if name == "" {
		return model.User{}, model.ErrInvalidUsername
	}
	if _, ok := m.usersByName[name]; ok {
		return model.User{}, model.ErrAlreadyExists
	}
	if u.ID == "" {
		u.ID = m.idGen(model.UserIDPrefix)
	}
	u.Username = name
	u.DisplayName = sanitizeDisplayName(u.DisplayName)
	if u.Status == "" {
		u.Status = model.UserActive
	}
	m.users[u.ID] = u
	m.usersByName[name] = u.ID
	m.persist()
	return u, nil
}

func (m *MemoryStore) GetUserByUsername(_ context.Context, username string) (model.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	id, ok := m.usersByName[strings.ToLower(strings.TrimSpace(username))]
	if !ok {
		return model.User{}, model.ErrNotFound
	}
	return m.users[id], nil
}

func (m *MemoryStore) GetUserByID(_ context.Context, id string) (model.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	u, ok := m.users[id]
	if !ok {
		return model.User{}, model.ErrNotFound
	}
	return u, nil
}

func (m *MemoryStore) ListUsers(_ context.Context, f model.UserFilter) ([]model.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]model.User, 0, len(m.users))
	for _, u := range m.users {
		if f.Role != "" && u.Role != f.Role {
			continue
		}
		if f.Status != "" && u.Status != f.Status {
			continue
		}
		if f.Query != "" && !matchQuery(u.Username+" "+u.DisplayName, f.Query) {
			continue
		}
		out = append(out, u)
	}
	return out, nil
}

func (m *MemoryStore) UpdateUser(_ context.Context, u model.User) (model.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	old, ok := m.users[u.ID]
	if !ok {
		return model.User{}, model.ErrNotFound
	}
	u.Username = old.Username
	u.DisplayName = sanitizeDisplayName(u.DisplayName)
	m.users[u.ID] = u
	m.persist()
	return u, nil
}

func (m *MemoryStore) CountUsers(_ context.Context) (total, active, frozen int, err error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, u := range m.users {
		total++
		switch u.Status {
		case model.UserActive:
			active++
		case model.UserFrozen:
			frozen++
		}
	}
	return total, active, frozen, nil
}

func (m *MemoryStore) CreateActivity(_ context.Context, a model.Activity) (model.Activity, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if a.ID == "" {
		a.ID = m.idGen(model.ActivityIDPrefix)
	}
	if a.CheckInCode == "" {
		a.CheckInCode = m.codeGen(6)
	}
	a.RequiredSkills = cloneStrings(a.RequiredSkills)
	m.activities[a.ID] = a
	m.persist()
	return a, nil
}

func (m *MemoryStore) GetActivity(_ context.Context, id string) (model.Activity, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	a, ok := m.activities[id]
	if !ok {
		return model.Activity{}, model.ErrNotFound
	}
	return a, nil
}

func (m *MemoryStore) ListActivities(_ context.Context, f model.ActivityFilter) ([]model.Activity, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]model.Activity, 0)
	for _, a := range m.activities {
		if !f.IncludeDraft && a.Status == model.ActivityDraft {
			if f.OrganizerID == "" || a.OrganizerID != f.OrganizerID {
				continue
			}
		}
		if f.Category != "" && a.Category != f.Category {
			continue
		}
		if f.Status != "" && a.Status != f.Status {
			continue
		}
		if f.OrganizerID != "" && a.OrganizerID != f.OrganizerID {
			continue
		}
		if f.TeamID != "" && a.TeamID != f.TeamID {
			continue
		}
		if !f.From.IsZero() && a.EndAt.Before(f.From) {
			continue
		}
		if !f.To.IsZero() && a.StartAt.After(f.To) {
			continue
		}
		if f.Query != "" && !matchQuery(a.Title+" "+a.Content+" "+a.Location, f.Query) {
			continue
		}
		out = append(out, a)
	}
	return out, nil
}

func (m *MemoryStore) UpdateActivity(_ context.Context, a model.Activity) (model.Activity, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.activities[a.ID]; !ok {
		return model.Activity{}, model.ErrNotFound
	}
	a.RequiredSkills = cloneStrings(a.RequiredSkills)
	m.activities[a.ID] = a
	m.persist()
	return a, nil
}

func (m *MemoryStore) CountActivitiesByOrganizerOpen(_ context.Context, organizerID string) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	n := 0
	for _, a := range m.activities {
		if a.OrganizerID != organizerID {
			continue
		}
		if a.Status.IsTerminal() {
			continue
		}
		n++
	}
	return n, nil
}

func (m *MemoryStore) CountActivitiesByStatus(_ context.Context) (map[model.ActivityStatus]int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := map[model.ActivityStatus]int{}
	for _, a := range m.activities {
		out[a.Status]++
	}
	return out, nil
}

func (m *MemoryStore) RecountActivityCounters(_ context.Context, activityID string) (model.Activity, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.activities[activityID]
	if !ok {
		return model.Activity{}, model.ErrNotFound
	}
	approved, wait := 0, 0
	for _, s := range m.signups {
		if s.ActivityID != activityID {
			continue
		}
		switch s.Status {
		case model.SignupApproved:
			approved++
		case model.SignupWaitlisted:
			wait++
		}
	}
	a.ApprovedCount = approved
	a.WaitlistCount = wait
	m.activities[activityID] = a
	m.persist()
	return a, nil
}

func (m *MemoryStore) CreateSignup(_ context.Context, s model.Signup) (model.Signup, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s.ID == "" {
		s.ID = m.idGen(model.SignupIDPrefix)
	}
	m.signups[s.ID] = s
	m.persist()
	return s, nil
}

func (m *MemoryStore) GetSignup(_ context.Context, id string) (model.Signup, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.signups[id]
	if !ok {
		return model.Signup{}, model.ErrNotFound
	}
	return s, nil
}

func (m *MemoryStore) GetSignupByActivityVolunteer(_ context.Context, activityID, volunteerID string) (model.Signup, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var found model.Signup
	ok := false
	for _, s := range m.signups {
		if s.ActivityID == activityID && s.VolunteerID == volunteerID {
			if !ok || s.CreatedAt.After(found.CreatedAt) {
				found = s
				ok = true
			}
		}
	}
	if !ok {
		return model.Signup{}, model.ErrNotFound
	}
	return found, nil
}

func (m *MemoryStore) ListSignupsByActivity(_ context.Context, activityID string) ([]model.Signup, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := []model.Signup{}
	for _, s := range m.signups {
		if s.ActivityID == activityID {
			out = append(out, s)
		}
	}
	return out, nil
}

func (m *MemoryStore) ListSignupsByVolunteer(_ context.Context, volunteerID string) ([]model.Signup, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := []model.Signup{}
	for _, s := range m.signups {
		if s.VolunteerID == volunteerID {
			out = append(out, s)
		}
	}
	return out, nil
}

func (m *MemoryStore) UpdateSignup(_ context.Context, s model.Signup) (model.Signup, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.signups[s.ID]; !ok {
		return model.Signup{}, model.ErrNotFound
	}
	m.signups[s.ID] = s
	m.persist()
	return s, nil
}

func (m *MemoryStore) CountApprovedByActivity(_ context.Context, activityID string) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	n := 0
	for _, s := range m.signups {
		if s.ActivityID == activityID && s.Status == model.SignupApproved {
			n++
		}
	}
	return n, nil
}

func (m *MemoryStore) CountWaitlistByActivity(_ context.Context, activityID string) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	n := 0
	for _, s := range m.signups {
		if s.ActivityID == activityID && s.Status == model.SignupWaitlisted {
			n++
		}
	}
	return n, nil
}

func (m *MemoryStore) NextWaitlistSeq(_ context.Context, activityID string) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	max := 0
	for _, s := range m.signups {
		if s.ActivityID == activityID && s.WaitlistSeq > max {
			max = s.WaitlistSeq
		}
	}
	return max + 1, nil
}

func (m *MemoryStore) ListApprovedOverlapping(_ context.Context, volunteerID string, start, end time.Time, excludeActivityID string) ([]model.Signup, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := []model.Signup{}
	for _, s := range m.signups {
		if s.VolunteerID != volunteerID || s.Status != model.SignupApproved {
			continue
		}
		if s.ActivityID == excludeActivityID {
			continue
		}
		act, ok := m.activities[s.ActivityID]
		if !ok || act.Status == model.ActivityCancelled {
			continue
		}
		if start.Before(act.EndAt) && act.StartAt.Before(end) {
			out = append(out, s)
		}
	}
	return out, nil
}

func (m *MemoryStore) CountNoShowsSince(_ context.Context, volunteerID string, since time.Time) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	n := 0
	for _, s := range m.signups {
		if s.VolunteerID != volunteerID || s.Status != model.SignupNoShow {
			continue
		}
		if s.UpdatedAt.Before(since) {
			continue
		}
		n++
	}
	return n, nil
}
