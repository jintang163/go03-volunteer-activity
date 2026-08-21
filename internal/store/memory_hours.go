package store

import (
	"context"
	"time"

	"go03-volunteer-activity/internal/model"
	"go03-volunteer-activity/internal/policy"
)

func (m *MemoryStore) CreateCheckIn(_ context.Context, c model.CheckIn) (model.CheckIn, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if c.ID == "" {
		c.ID = m.idGen(model.CheckInIDPrefix)
	}
	m.checkins[c.ID] = c
	m.persist()
	return c, nil
}

// ReserveCheckIn 在单次写锁内原子地完成「检查同一活动同一志愿者是否已签到 + 写入记录」。
// 它消除 SelfCheckIn/ProxyCheckIn 此前「先 GetCheckInByActivityVolunteer 检查、后 CreateCheckIn
// 写入」两步之间的 TOCTOU 竞态：两个并发签到请求都通过存在性检查后，第二个进入写锁时
// 重新读到已写入的签到记录，即返回 ErrAlreadyCheckedIn，从而保证同一志愿者在同一活动只能
// 成功签到一次，避免重复生成签到记录。
func (m *MemoryStore) ReserveCheckIn(_ context.Context, c model.CheckIn) (model.CheckIn, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, ex := range m.checkins {
		if ex.ActivityID == c.ActivityID && ex.VolunteerID == c.VolunteerID {
			return model.CheckIn{}, model.ErrAlreadyCheckedIn
		}
	}
	if c.ID == "" {
		c.ID = m.idGen(model.CheckInIDPrefix)
	}
	m.checkins[c.ID] = c
	m.persist()
	return c, nil
}

func (m *MemoryStore) GetCheckIn(_ context.Context, id string) (model.CheckIn, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	c, ok := m.checkins[id]
	if !ok {
		return model.CheckIn{}, model.ErrNotFound
	}
	return c, nil
}

func (m *MemoryStore) GetCheckInByActivityVolunteer(_ context.Context, activityID, volunteerID string) (model.CheckIn, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, c := range m.checkins {
		if c.ActivityID == activityID && c.VolunteerID == volunteerID {
			return c, nil
		}
	}
	return model.CheckIn{}, model.ErrNotFound
}

func (m *MemoryStore) ListCheckInsByActivity(_ context.Context, activityID string) ([]model.CheckIn, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := []model.CheckIn{}
	for _, c := range m.checkins {
		if c.ActivityID == activityID {
			out = append(out, c)
		}
	}
	return out, nil
}

func (m *MemoryStore) UpdateCheckIn(_ context.Context, c model.CheckIn) (model.CheckIn, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.checkins[c.ID]; !ok {
		return model.CheckIn{}, model.ErrNotFound
	}
	m.checkins[c.ID] = c
	m.persist()
	return c, nil
}

// ReserveCheckOut 在单次写锁内原子地完成「校验签到记录尚未签退 + 写入签退时间」。
// 它消除 CheckOut 此前「先 GetCheckInByActivityVolunteer 检查 HasCheckedOut、后
// UpdateCheckIn 写入」两步之间的 TOCTOU 竞态：同一志愿者的两个并发签退请求都读到
// 未签退状态而通过检查后，第二个进入写锁时重新读到记录已带签退时间，即返回
// ErrAlreadyCheckedOut，从而保证同一签到记录的签退状态只能迁移一次，且签退后的
// 工时草拟只随胜出请求触发一次。
func (m *MemoryStore) ReserveCheckOut(_ context.Context, c model.CheckIn) (model.CheckIn, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	current, ok := m.checkins[c.ID]
	if !ok {
		return model.CheckIn{}, model.ErrNotFound
	}
	if current.HasCheckedOut() {
		return current, model.ErrAlreadyCheckedOut
	}
	m.checkins[c.ID] = c
	m.persist()
	return c, nil
}

func (m *MemoryStore) CountCheckInsOnDay(_ context.Context, day time.Time) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	n := 0
	for _, c := range m.checkins {
		if sameDay(c.CheckInAt, day) {
			n++
		}
	}
	return n, nil
}

func (m *MemoryStore) CreateHour(_ context.Context, h model.HourRecord) (model.HourRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if h.ID == "" {
		h.ID = m.idGen(model.HourIDPrefix)
	}
	m.hours[h.ID] = h
	m.persist()
	return h, nil
}

func (m *MemoryStore) GetHour(_ context.Context, id string) (model.HourRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	h, ok := m.hours[id]
	if !ok {
		return model.HourRecord{}, model.ErrNotFound
	}
	return h, nil
}

func (m *MemoryStore) GetOpenHour(_ context.Context, activityID, volunteerID string) (model.HourRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, h := range m.hours {
		if h.ActivityID == activityID && h.VolunteerID == volunteerID {
			if h.Status == model.HourPending || h.Status == model.HourDraft || h.Status == model.HourApproved {
				return h, nil
			}
		}
	}
	return model.HourRecord{}, model.ErrNotFound
}

func (m *MemoryStore) ListHoursByVolunteer(_ context.Context, volunteerID string) ([]model.HourRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := []model.HourRecord{}
	for _, h := range m.hours {
		if h.VolunteerID == volunteerID {
			out = append(out, h)
		}
	}
	return out, nil
}

func (m *MemoryStore) ListHoursByActivity(_ context.Context, activityID string) ([]model.HourRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := []model.HourRecord{}
	for _, h := range m.hours {
		if h.ActivityID == activityID {
			out = append(out, h)
		}
	}
	return out, nil
}

func (m *MemoryStore) ListPendingHours(_ context.Context) ([]model.HourRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := []model.HourRecord{}
	for _, h := range m.hours {
		if h.Status == model.HourPending || h.Status == model.HourDraft {
			out = append(out, h)
		}
	}
	return out, nil
}

func (m *MemoryStore) UpdateHour(_ context.Context, h model.HourRecord) (model.HourRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.hours[h.ID]; !ok {
		return model.HourRecord{}, model.ErrNotFound
	}
	m.hours[h.ID] = h
	m.persist()
	return h, nil
}

func (m *MemoryStore) CreateHourLedger(_ context.Context, l model.HourLedger) (model.HourLedger, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if l.ID == "" {
		l.ID = m.idGen(model.HourLedgerIDPrefix)
	}
	m.hourLedgers[l.ID] = l
	m.persist()
	return l, nil
}

func (m *MemoryStore) ListHourLedgers(_ context.Context, userID string) ([]model.HourLedger, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := []model.HourLedger{}
	for _, l := range m.hourLedgers {
		if l.UserID == userID {
			out = append(out, l)
		}
	}
	return out, nil
}

func (m *MemoryStore) CreatePointLedger(_ context.Context, l model.PointLedger) (model.PointLedger, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if l.ID == "" {
		l.ID = m.idGen(model.PointLedgerIDPrefix)
	}
	m.pointLedgers[l.ID] = l
	m.persist()
	return l, nil
}

func (m *MemoryStore) ListPointLedgers(_ context.Context, userID string) ([]model.PointLedger, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := []model.PointLedger{}
	for _, l := range m.pointLedgers {
		if l.UserID == userID {
			out = append(out, l)
		}
	}
	return out, nil
}

// ApplyHourApproval 在单次写锁内原子地完成「校验工时仍在待审 + 入账工时与积分 +
// 写两条流水」。它消除 Approve 此前「先 GetHour 读状态、后 ApplyHoursAndPoints 入账」
// 两步之间的 TOCTOU 竞态：两个并发审批请求都读到 pending 通过校验后，第二个进入写锁时
// 重新读到工时已是 approved，即返回 ErrHoursNotPending，从而保证同一条工时只能成功入账
// 一次，避免重复累加工时、积分与重复生成账本。
func (m *MemoryStore) ApplyHourApproval(_ context.Context, hour model.HourRecord, hl model.HourLedger, pl model.PointLedger) (model.HourRecord, model.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	current, ok := m.hours[hour.ID]
	if !ok {
		return model.HourRecord{}, model.User{}, model.ErrNotFound
	}
	if current.Status != model.HourPending && current.Status != model.HourDraft {
		return model.HourRecord{}, model.User{}, model.ErrHoursNotPending
	}
	u, ok := m.users[hour.VolunteerID]
	if !ok {
		return model.HourRecord{}, model.User{}, model.ErrNotFound
	}
	u.TotalMinutes += hl.DeltaMin
	if u.TotalMinutes < 0 {
		u.TotalMinutes = 0
	}
	u.Points += pl.Delta
	if u.Points < 0 {
		u.Points = 0
	}
	u.UpdatedAt = hour.UpdatedAt
	hl.BalanceMin = u.TotalMinutes
	pl.Balance = u.Points
	if hl.ID == "" {
		hl.ID = m.idGen(model.HourLedgerIDPrefix)
	}
	if pl.ID == "" {
		pl.ID = m.idGen(model.PointLedgerIDPrefix)
	}
	m.hours[hour.ID] = hour
	m.users[u.ID] = u
	m.hourLedgers[hl.ID] = hl
	m.pointLedgers[pl.ID] = pl
	m.persist()
	return hour, u, nil
}

func (m *MemoryStore) ApplyPoints(_ context.Context, user model.User, pl model.PointLedger) (model.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[user.ID]
	if !ok {
		return model.User{}, model.ErrNotFound
	}
	u.Points += pl.Delta
	if u.Points < 0 {
		u.Points = 0
	}
	if pl.Reason == model.PointNoShow {
		u.NoShowCount++
	}
	u.UpdatedAt = time.Now()
	pl.Balance = u.Points
	if pl.ID == "" {
		pl.ID = m.idGen(model.PointLedgerIDPrefix)
	}
	m.users[u.ID] = u
	m.pointLedgers[pl.ID] = pl
	m.persist()
	return u, nil
}

func PointsFor(minutes int) int {
	return policy.PointsForMinutes(minutes)
}
