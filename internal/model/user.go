package model

import (
	"strings"
	"time"
)

type UserRole string

const (
	RoleAdmin      UserRole = "admin"
	RoleOrganizer  UserRole = "organizer"
	RoleVolunteer  UserRole = "volunteer"
)

func ParseUserRole(s string) (UserRole, bool) {
	switch UserRole(strings.ToLower(strings.TrimSpace(s))) {
	case RoleAdmin:
		return RoleAdmin, true
	case RoleOrganizer:
		return RoleOrganizer, true
	case RoleVolunteer:
		return RoleVolunteer, true
	default:
		return "", false
	}
}

type UserStatus string

const (
	UserActive UserStatus = "active"
	UserFrozen UserStatus = "frozen"
	UserBanned UserStatus = "banned"
)

type User struct {
	ID            string     `json:"id"`
	Username      string     `json:"username"`
	DisplayName   string     `json:"display_name"`
	Role          UserRole   `json:"role"`
	Status        UserStatus `json:"status"`
	PasswordSalt  string     `json:"password_salt"`
	PasswordHash  string     `json:"password_hash"`
	Iterations    int        `json:"iterations"`
	Phone         string     `json:"phone"`
	Bio           string     `json:"bio"`
	Age           int        `json:"age"`
	Skills        []string   `json:"skills"`
	TotalMinutes  int        `json:"total_minutes"`
	Points        int        `json:"points"`
	NoShowCount   int        `json:"no_show_count"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

func (u User) IsAdmin() bool     { return u.Role == RoleAdmin }
func (u User) IsOrganizer() bool { return u.Role == RoleOrganizer || u.Role == RoleAdmin }
func (u User) IsVolunteer() bool { return u.Role == RoleVolunteer }
func (u User) IsBanned() bool    { return u.Status == UserBanned }
func (u User) IsFrozen() bool    { return u.Status == UserFrozen }

func (u User) CanWrite() error {
	if u.IsBanned() {
		return ErrAccountBanned
	}
	if u.IsFrozen() {
		return ErrAccountFrozen
	}
	return nil
}

func (u User) Public() PublicUser {
	return PublicUser{
		ID:           u.ID,
		Username:     u.Username,
		DisplayName:  u.DisplayName,
		Role:         u.Role,
		Status:       u.Status,
		Phone:        u.Phone,
		Bio:          u.Bio,
		Age:          u.Age,
		Skills:       append([]string(nil), u.Skills...),
		TotalMinutes: u.TotalMinutes,
		Points:       u.Points,
		NoShowCount:  u.NoShowCount,
		CreatedAt:    u.CreatedAt,
	}
}

func (u User) Safe() SafeUser {
	p := u.Public()
	return SafeUser{PublicUser: p}
}

func (u User) HasSkill(skill string) bool {
	skill = strings.ToLower(strings.TrimSpace(skill))
	if skill == "" {
		return true
	}
	for _, s := range u.Skills {
		if strings.ToLower(strings.TrimSpace(s)) == skill {
			return true
		}
	}
	return false
}

func (u User) HasAllSkills(required []string) bool {
	for _, r := range required {
		if strings.TrimSpace(r) == "" {
			continue
		}
		if !u.HasSkill(r) {
			return false
		}
	}
	return true
}

type PublicUser struct {
	ID           string     `json:"id"`
	Username     string     `json:"username"`
	DisplayName  string     `json:"display_name"`
	Role         UserRole   `json:"role"`
	Status       UserStatus `json:"status"`
	Phone        string     `json:"phone,omitempty"`
	Bio          string     `json:"bio,omitempty"`
	Age          int        `json:"age,omitempty"`
	Skills       []string   `json:"skills,omitempty"`
	TotalMinutes int        `json:"total_minutes"`
	Points       int        `json:"points"`
	NoShowCount  int        `json:"no_show_count"`
	CreatedAt    time.Time  `json:"created_at"`
}

type SafeUser struct {
	PublicUser
}

type UserFilter struct {
	Query  string
	Role   UserRole
	Status UserStatus
}
