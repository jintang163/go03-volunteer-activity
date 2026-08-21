package model

import "time"

type Team struct {
	ID          string    `json:"id"`
	OwnerID     string    `json:"owner_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type TeamMember struct {
	ID        string    `json:"id"`
	TeamID    string    `json:"team_id"`
	UserID    string    `json:"user_id"`
	Role      string    `json:"role"`
	JoinedAt  time.Time `json:"joined_at"`
}

type NotificationKind string

const (
	NotifySignupResult NotificationKind = "signup_result"
	NotifyWaitlistUp   NotificationKind = "waitlist_promoted"
	NotifyHoursResult  NotificationKind = "hours_result"
	NotifyCertificate  NotificationKind = "certificate"
	NotifyActivity     NotificationKind = "activity"
	NotifyNoShow       NotificationKind = "no_show"
)

type Notification struct {
	ID        string           `json:"id"`
	UserID    string           `json:"user_id"`
	Kind      NotificationKind `json:"kind"`
	Title     string           `json:"title"`
	Body      string           `json:"body"`
	RelatedID string           `json:"related_id"`
	Read      bool             `json:"read"`
	CreatedAt time.Time        `json:"created_at"`
	ReadAt    *time.Time       `json:"read_at,omitempty"`
}

type AuditAction string

const (
	AuditPublish       AuditAction = "publish"
	AuditCancelActivity AuditAction = "cancel_activity"
	AuditApproveSignup AuditAction = "approve_signup"
	AuditRejectSignup  AuditAction = "reject_signup"
	AuditCheckIn       AuditAction = "check_in"
	AuditCheckOut      AuditAction = "check_out"
	AuditHoursApprove  AuditAction = "hours_approve"
	AuditHoursReject   AuditAction = "hours_reject"
	AuditIssueCert     AuditAction = "issue_certificate"
	AuditFreezeUser    AuditAction = "freeze_user"
)

type AuditLog struct {
	ID         string      `json:"id"`
	ActorID    string      `json:"actor_id"`
	Action     AuditAction `json:"action"`
	TargetType string      `json:"target_type"`
	TargetID   string      `json:"target_id"`
	Detail     string      `json:"detail"`
	CreatedAt  time.Time   `json:"created_at"`
}

type HealthResponse struct {
	Status string `json:"status"`
}

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type LoginResponse struct {
	Token string     `json:"token"`
	User  PublicUser `json:"user"`
}

type MeResponse struct {
	User             PublicUser `json:"user"`
	UnreadNotify     int        `json:"unread_notifications"`
	NextCertTier     string     `json:"next_certificate_tier"`
	MinutesToNextCert int       `json:"minutes_to_next_certificate"`
}

type StatsSnapshot struct {
	UsersTotal       int            `json:"users_total"`
	UsersActive      int            `json:"users_active"`
	ActivitiesByStatus map[string]int `json:"activities_by_status"`
	SignupsApproved  int            `json:"signups_approved"`
	SignupsWaitlist  int            `json:"signups_waitlist"`
	CheckInsToday    int            `json:"checkins_today"`
	HoursThisMonth   int            `json:"hours_this_month_minutes"`
	NoShowRateBP     int            `json:"no_show_rate_basis_points"`
	PendingHours     int            `json:"pending_hours"`
}

type OrganizerStats struct {
	MyActivities   int `json:"my_activities"`
	ApprovedPeople int `json:"approved_people"`
	CheckedIn      int `json:"checked_in"`
	PendingHours   int `json:"pending_hours"`
}
