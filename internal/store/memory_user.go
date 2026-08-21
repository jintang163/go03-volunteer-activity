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

// ReserveSignup 在单次写锁内原子地完成录取判定并写入报名记录。
// 它把 decideStatus 的「检查容量/候补/冲突/审批」与 CreateSignup 的写入合并为
// 一个不可分割的临界区：并发请求串行进入写锁，第二个请求在锁内重新计数时看到
// 已达上限即返回 ErrCapacityFull，从而保证同一时刻最多一个请求成功录取。
func (m *MemoryStore) ReserveSignup(_ context.Context, s model.Signup, act model.Activity) (model.Signup, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	approved, wait := 0, 0
	for _, ex := range m.signups {
		if ex.ActivityID != act.ID {
			continue
		}
		switch ex.Status {
		case model.SignupApproved:
			approved++
		case model.SignupWaitlisted:
			wait++
		}
	}

	if approved >= act.Capacity {
		if !act.WaitlistEnabled {
			return model.Signup{}, model.ErrCapacityFull
		}
		if wait >= act.WaitlistLimit {
			return model.Signup{}, model.ErrWaitlistFull
		}
		// 候补席：跳过时间冲突判定（候补可在递补时再校验），直接分配序号写入。
		s.Status = model.SignupWaitlisted
		if s.WaitlistSeq == 0 {
			s.WaitlistSeq = m.nextWaitlistSeqLocked(act.ID)
		}
	} else {
		// 时间冲突判定：与该志愿者已被录取的其他活动在时间上是否重叠。
		overlap := false
		for _, ex := range m.signups {
			if ex.VolunteerID != s.VolunteerID || ex.Status != model.SignupApproved {
				continue
			}
			if ex.ActivityID == act.ID {
				continue
			}
			other, ok := m.activities[ex.ActivityID]
			if !ok || other.Status == model.ActivityCancelled {
				continue
			}
			if act.StartAt.Before(other.EndAt) && other.StartAt.Before(act.EndAt) {
				overlap = true
				break
			}
		}
		if overlap {
			if act.WaitlistEnabled {
				if wait >= act.WaitlistLimit {
					return model.Signup{}, model.ErrWaitlistFull
				}
				s.Status = model.SignupWaitlisted
				if s.WaitlistSeq == 0 {
					s.WaitlistSeq = m.nextWaitlistSeqLocked(act.ID)
				}
			} else {
				return model.Signup{}, model.ErrScheduleConflict
			}
		} else if act.NeedApproval {
			s.Status = model.SignupPending
		} else {
			s.Status = model.SignupApproved
			t := s.CreatedAt
			s.ApprovedAt = &t
		}
	}

	if s.ID == "" {
		s.ID = m.idGen(model.SignupIDPrefix)
	}
	m.signups[s.ID] = s
	m.persist()
	return s, nil
}

// nextWaitlistSeqLocked 返回该活动下一个候补序号。调用方必须已持有 m.mu。
func (m *MemoryStore) nextWaitlistSeqLocked(activityID string) int {
	max := 0
	for _, s := range m.signups {
		if s.ActivityID == activityID && s.WaitlistSeq > max {
			max = s.WaitlistSeq
		}
	}
	return max + 1
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

// ReserveSignupApproval 在单次写锁内原子地完成「校验报名仍待审批 + 容量是否已满 +
// 时间是否冲突 + 写入已录取状态」。它消除 Approve 此前「先 CountApprovedByActivity 检查容量、
// ListApprovedOverlapping 检查冲突、后 UpdateSignup 写入」两步之间的 TOCTOU 竞态：两个并发
// 审批请求都读到 approved < capacity 而通过检查后，第二个进入写锁时重新计数发现已达上限即
// 返回 ErrCapacityFull，从而保证最后一个名额只能被一条报名成功占用，且活动录取计数保持一致。
func (m *MemoryStore) ReserveSignupApproval(_ context.Context, s model.Signup, act model.Activity) (model.Signup, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	current, ok := m.signups[s.ID]
	if !ok {
		return model.Signup{}, model.ErrNotFound
	}
	// 锁内重新校验状态：仅 pending/waitlisted 可录取，已被并发审批处理过的报名直接报冲突。
	if current.Status != model.SignupPending && current.Status != model.SignupWaitlisted {
		return current, model.ErrConflict
	}

	// 锁内重新计数已录取人数：与 ReserveSignup 同一判定口径，第二个请求进入时名额已被占用。
	approved := 0
	for _, ex := range m.signups {
		if ex.ActivityID == act.ID && ex.Status == model.SignupApproved {
			approved++
		}
	}
	if approved >= act.Capacity {
		return current, model.ErrCapacityFull
	}

	// 锁内重新校验时间冲突：与本志愿者已被录取的其他活动在时间上是否重叠。
	for _, ex := range m.signups {
		if ex.VolunteerID != s.VolunteerID || ex.Status != model.SignupApproved {
			continue
		}
		if ex.ActivityID == act.ID {
			continue
		}
		other, ok := m.activities[ex.ActivityID]
		if !ok || other.Status == model.ActivityCancelled {
			continue
		}
		if act.StartAt.Before(other.EndAt) && other.StartAt.Before(act.EndAt) {
			return current, model.ErrScheduleConflict
		}
	}

	m.signups[s.ID] = s
	m.persist()
	return s, nil
}

// ReserveSignupCancellation 在单次写锁内原子地完成「校验报名仍处于有效状态 + 写入已取消状态」。
// 它消除 Cancel 此前「先 GetSignup 读取状态、IsActive 检查、后 UpdateSignup 写入」两步之间的
// TOCTOU 竞态：同一条已录取报名在活动开始后的两个并发取消请求都读到有效状态而通过检查后，
// 第二个进入写锁时重新读到报名已是 cancelled，即返回 ErrConflict，从而保证同一报名的取消状态
// 只能迁移一次，且迟到扣分、候补递补等后续副作用只随胜出请求触发一次。
func (m *MemoryStore) ReserveSignupCancellation(_ context.Context, s model.Signup, act model.Activity) (model.Signup, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	current, ok := m.signups[s.ID]
	if !ok {
		return model.Signup{}, model.ErrNotFound
	}
	// 锁内重新校验状态：仅有效状态（pending/approved/waitlisted）可取消，已被并发取消处理过的
	// 报名直接返回当前记录 + ErrConflict，使服务层据此跳过迟到扣分与候补递补等副作用。
	if !current.Status.IsActive() {
		return current, model.ErrConflict
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
