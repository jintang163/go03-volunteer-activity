package store

import (
	"sync"

	"go03-volunteer-activity/internal/model"
)

type Snapshot struct {
	Users         []model.User         `json:"users"`
	Activities    []model.Activity     `json:"activities"`
	Signups       []model.Signup       `json:"signups"`
	CheckIns      []model.CheckIn      `json:"checkins"`
	Hours         []model.HourRecord   `json:"hours"`
	HourLedgers   []model.HourLedger   `json:"hour_ledgers"`
	PointLedgers  []model.PointLedger  `json:"point_ledgers"`
	Teams         []model.Team         `json:"teams"`
	TeamMembers   []model.TeamMember   `json:"team_members"`
	Notifications []model.Notification `json:"notifications"`
	Audits        []model.AuditLog     `json:"audits"`
	Certificates  []model.Certificate  `json:"certificates"`
	Feedbacks     []model.Feedback     `json:"feedbacks"`
}

type MemoryStore struct {
	mu     sync.RWMutex
	hook   func()
	idGen  func(prefix string) string
	codeGen func(n int) string

	users         map[string]model.User
	usersByName   map[string]string
	activities    map[string]model.Activity
	signups       map[string]model.Signup
	checkins      map[string]model.CheckIn
	hours         map[string]model.HourRecord
	hourLedgers   map[string]model.HourLedger
	pointLedgers  map[string]model.PointLedger
	teams         map[string]model.Team
	teamMembers   map[string]model.TeamMember
	notifications map[string]model.Notification
	audits        map[string]model.AuditLog
	certificates  map[string]model.Certificate
	feedbacks     map[string]model.Feedback
}

func NewMemoryStore(idGen func(string) string, persist func()) *MemoryStore {
	if idGen == nil {
		idGen = defaultIDGenerator
	}
	m := &MemoryStore{
		hook:          persist,
		idGen:         idGen,
		codeGen:       randomCode,
		users:         map[string]model.User{},
		usersByName:   map[string]string{},
		activities:    map[string]model.Activity{},
		signups:       map[string]model.Signup{},
		checkins:      map[string]model.CheckIn{},
		hours:         map[string]model.HourRecord{},
		hourLedgers:   map[string]model.HourLedger{},
		pointLedgers:  map[string]model.PointLedger{},
		teams:         map[string]model.Team{},
		teamMembers:   map[string]model.TeamMember{},
		notifications: map[string]model.Notification{},
		audits:        map[string]model.AuditLog{},
		certificates:  map[string]model.Certificate{},
		feedbacks:     map[string]model.Feedback{},
	}
	return m
}

func (m *MemoryStore) SetPersistHook(fn func()) { m.hook = fn }

func (m *MemoryStore) persist() {
	if m.hook != nil {
		m.hook()
	}
}

func (m *MemoryStore) NewID(prefix string) string { return m.idGen(prefix) }

func (m *MemoryStore) NewCheckInCode() string { return m.codeGen(6) }

func (m *MemoryStore) Snapshot() Snapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.snapshotNoLock()
}

// snapshotNoLock 假定调用方已持有 mu（写锁或读锁）。persist 钩子在写路径持锁时调用，不可再 RLock。
func (m *MemoryStore) snapshotNoLock() Snapshot {
	s := Snapshot{}
	for _, v := range m.users {
		s.Users = append(s.Users, v)
	}
	for _, v := range m.activities {
		s.Activities = append(s.Activities, v)
	}
	for _, v := range m.signups {
		s.Signups = append(s.Signups, v)
	}
	for _, v := range m.checkins {
		s.CheckIns = append(s.CheckIns, v)
	}
	for _, v := range m.hours {
		s.Hours = append(s.Hours, v)
	}
	for _, v := range m.hourLedgers {
		s.HourLedgers = append(s.HourLedgers, v)
	}
	for _, v := range m.pointLedgers {
		s.PointLedgers = append(s.PointLedgers, v)
	}
	for _, v := range m.teams {
		s.Teams = append(s.Teams, v)
	}
	for _, v := range m.teamMembers {
		s.TeamMembers = append(s.TeamMembers, v)
	}
	for _, v := range m.notifications {
		s.Notifications = append(s.Notifications, v)
	}
	for _, v := range m.audits {
		s.Audits = append(s.Audits, v)
	}
	for _, v := range m.certificates {
		s.Certificates = append(s.Certificates, v)
	}
	for _, v := range m.feedbacks {
		s.Feedbacks = append(s.Feedbacks, v)
	}
	return s
}

func (m *MemoryStore) ReplaceAll(s Snapshot) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.users = map[string]model.User{}
	m.usersByName = map[string]string{}
	m.activities = map[string]model.Activity{}
	m.signups = map[string]model.Signup{}
	m.checkins = map[string]model.CheckIn{}
	m.hours = map[string]model.HourRecord{}
	m.hourLedgers = map[string]model.HourLedger{}
	m.pointLedgers = map[string]model.PointLedger{}
	m.teams = map[string]model.Team{}
	m.teamMembers = map[string]model.TeamMember{}
	m.notifications = map[string]model.Notification{}
	m.audits = map[string]model.AuditLog{}
	m.certificates = map[string]model.Certificate{}
	m.feedbacks = map[string]model.Feedback{}
	for _, v := range s.Users {
		m.users[v.ID] = v
		m.usersByName[v.Username] = v.ID
	}
	for _, v := range s.Activities {
		m.activities[v.ID] = v
	}
	for _, v := range s.Signups {
		m.signups[v.ID] = v
	}
	for _, v := range s.CheckIns {
		m.checkins[v.ID] = v
	}
	for _, v := range s.Hours {
		m.hours[v.ID] = v
	}
	for _, v := range s.HourLedgers {
		m.hourLedgers[v.ID] = v
	}
	for _, v := range s.PointLedgers {
		m.pointLedgers[v.ID] = v
	}
	for _, v := range s.Teams {
		m.teams[v.ID] = v
	}
	for _, v := range s.TeamMembers {
		m.teamMembers[v.ID] = v
	}
	for _, v := range s.Notifications {
		m.notifications[v.ID] = v
	}
	for _, v := range s.Audits {
		m.audits[v.ID] = v
	}
	for _, v := range s.Certificates {
		m.certificates[v.ID] = v
	}
	for _, v := range s.Feedbacks {
		m.feedbacks[v.ID] = v
	}
}
