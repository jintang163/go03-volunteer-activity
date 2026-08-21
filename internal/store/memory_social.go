package store

import (
	"context"
	"time"

	"go03-volunteer-activity/internal/model"
)

func (m *MemoryStore) CreateTeam(_ context.Context, t model.Team) (model.Team, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if t.ID == "" {
		t.ID = m.idGen(model.TeamIDPrefix)
	}
	m.teams[t.ID] = t
	m.persist()
	return t, nil
}

func (m *MemoryStore) GetTeam(_ context.Context, id string) (model.Team, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	t, ok := m.teams[id]
	if !ok {
		return model.Team{}, model.ErrNotFound
	}
	return t, nil
}

func (m *MemoryStore) ListTeams(_ context.Context, ownerID string) ([]model.Team, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := []model.Team{}
	for _, t := range m.teams {
		if ownerID == "" || t.OwnerID == ownerID {
			out = append(out, t)
		}
	}
	return out, nil
}

func (m *MemoryStore) UpdateTeam(_ context.Context, t model.Team) (model.Team, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.teams[t.ID]; !ok {
		return model.Team{}, model.ErrNotFound
	}
	m.teams[t.ID] = t
	m.persist()
	return t, nil
}

func (m *MemoryStore) CreateTeamMember(_ context.Context, mem model.TeamMember) (model.TeamMember, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if mem.ID == "" {
		mem.ID = m.idGen(model.MemberIDPrefix)
	}
	m.teamMembers[mem.ID] = mem
	m.persist()
	return mem, nil
}

func (m *MemoryStore) ListTeamMembers(_ context.Context, teamID string) ([]model.TeamMember, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := []model.TeamMember{}
	for _, mem := range m.teamMembers {
		if mem.TeamID == teamID {
			out = append(out, mem)
		}
	}
	return out, nil
}

func (m *MemoryStore) GetTeamMember(_ context.Context, teamID, userID string) (model.TeamMember, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, mem := range m.teamMembers {
		if mem.TeamID == teamID && mem.UserID == userID {
			return mem, nil
		}
	}
	return model.TeamMember{}, model.ErrNotFound
}

func (m *MemoryStore) DeleteTeamMember(_ context.Context, teamID, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, mem := range m.teamMembers {
		if mem.TeamID == teamID && mem.UserID == userID {
			delete(m.teamMembers, id)
			m.persist()
			return nil
		}
	}
	return model.ErrNotFound
}

func (m *MemoryStore) CreateNotification(_ context.Context, n model.Notification) (model.Notification, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if n.ID == "" {
		n.ID = m.idGen(model.NotificationIDPrefix)
	}
	m.notifications[n.ID] = n
	m.persist()
	return n, nil
}

func (m *MemoryStore) ListNotifications(_ context.Context, userID string, unreadOnly bool) ([]model.Notification, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := []model.Notification{}
	for _, n := range m.notifications {
		if n.UserID != userID {
			continue
		}
		if unreadOnly && n.Read {
			continue
		}
		out = append(out, n)
	}
	return out, nil
}

func (m *MemoryStore) GetNotification(_ context.Context, id string) (model.Notification, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	n, ok := m.notifications[id]
	if !ok {
		return model.Notification{}, model.ErrNotFound
	}
	return n, nil
}

func (m *MemoryStore) UpdateNotification(_ context.Context, n model.Notification) (model.Notification, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.notifications[n.ID]; !ok {
		return model.Notification{}, model.ErrNotFound
	}
	m.notifications[n.ID] = n
	m.persist()
	return n, nil
}

func (m *MemoryStore) MarkAllNotificationsRead(_ context.Context, userID string, at time.Time) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for id, item := range m.notifications {
		if item.UserID == userID && !item.Read {
			item.Read = true
			t := at
			item.ReadAt = &t
			m.notifications[id] = item
			n++
		}
	}
	if n > 0 {
		m.persist()
	}
	return n, nil
}

func (m *MemoryStore) CountUnreadNotifications(_ context.Context, userID string) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	n := 0
	for _, item := range m.notifications {
		if item.UserID == userID && !item.Read {
			n++
		}
	}
	return n, nil
}

func (m *MemoryStore) CreateAudit(_ context.Context, a model.AuditLog) (model.AuditLog, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if a.ID == "" {
		a.ID = m.idGen(model.AuditIDPrefix)
	}
	m.audits[a.ID] = a
	m.persist()
	return a, nil
}

func (m *MemoryStore) ListAudits(_ context.Context, targetType, targetID string) ([]model.AuditLog, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := []model.AuditLog{}
	for _, a := range m.audits {
		if targetType != "" && a.TargetType != targetType {
			continue
		}
		if targetID != "" && a.TargetID != targetID {
			continue
		}
		out = append(out, a)
	}
	return out, nil
}

func (m *MemoryStore) CreateCertificate(_ context.Context, c model.Certificate) (model.Certificate, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if c.ID == "" {
		c.ID = m.idGen(model.CertIDPrefix)
	}
	if c.Code == "" {
		c.Code = "VA-" + m.codeGen(10)
	}
	m.certificates[c.ID] = c
	m.persist()
	return c, nil
}

func (m *MemoryStore) GetCertificateByCode(_ context.Context, code string) (model.Certificate, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, c := range m.certificates {
		if c.Code == code {
			return c, nil
		}
	}
	return model.Certificate{}, model.ErrNotFound
}

func (m *MemoryStore) ListCertificatesByUser(_ context.Context, userID string) ([]model.Certificate, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := []model.Certificate{}
	for _, c := range m.certificates {
		if c.UserID == userID {
			out = append(out, c)
		}
	}
	return out, nil
}

func (m *MemoryStore) HasCertificateTier(_ context.Context, userID string, tier model.CertificateTier) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, c := range m.certificates {
		if c.UserID == userID && c.Tier == tier {
			return true, nil
		}
	}
	return false, nil
}

func (m *MemoryStore) CreateFeedback(_ context.Context, f model.Feedback) (model.Feedback, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if f.ID == "" {
		f.ID = m.idGen(model.FeedbackIDPrefix)
	}
	m.feedbacks[f.ID] = f
	m.persist()
	return f, nil
}

// ReserveFeedback 在单次写锁内原子地完成「检查同一活动同一志愿者是否已反馈 + 写入记录」。
// 它消除 SubmitFeedback 此前「先 GetFeedback 检查、后 CreateFeedback 写入」两步之间的
// TOCTOU 竞态：两个并发反馈请求都通过存在性检查后，第二个进入写锁时重新读到已写入的
// 反馈记录，即返回 ErrAlreadyFeedback，从而保证同一志愿者在同一活动只能成功反馈一次，
// 避免活动最终保存多条反馈。
func (m *MemoryStore) ReserveFeedback(_ context.Context, f model.Feedback) (model.Feedback, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, ex := range m.feedbacks {
		if ex.ActivityID == f.ActivityID && ex.VolunteerID == f.VolunteerID {
			return model.Feedback{}, model.ErrAlreadyFeedback
		}
	}
	if f.ID == "" {
		f.ID = m.idGen(model.FeedbackIDPrefix)
	}
	m.feedbacks[f.ID] = f
	m.persist()
	return f, nil
}

func (m *MemoryStore) GetFeedback(_ context.Context, activityID, volunteerID string) (model.Feedback, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, f := range m.feedbacks {
		if f.ActivityID == activityID && f.VolunteerID == volunteerID {
			return f, nil
		}
	}
	return model.Feedback{}, model.ErrNotFound
}

func (m *MemoryStore) ListFeedbackByActivity(_ context.Context, activityID string) ([]model.Feedback, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := []model.Feedback{}
	for _, f := range m.feedbacks {
		if f.ActivityID == activityID {
			out = append(out, f)
		}
	}
	return out, nil
}
