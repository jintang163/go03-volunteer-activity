package store

import (
	"context"
	"time"

	"go03-volunteer-activity/internal/model"
)

type Store interface {
	CreateUser(ctx context.Context, u model.User) (model.User, error)
	GetUserByUsername(ctx context.Context, username string) (model.User, error)
	GetUserByID(ctx context.Context, id string) (model.User, error)
	ListUsers(ctx context.Context, f model.UserFilter) ([]model.User, error)
	UpdateUser(ctx context.Context, u model.User) (model.User, error)
	CountUsers(ctx context.Context) (total, active, frozen int, err error)

	CreateActivity(ctx context.Context, a model.Activity) (model.Activity, error)
	GetActivity(ctx context.Context, id string) (model.Activity, error)
	ListActivities(ctx context.Context, f model.ActivityFilter) ([]model.Activity, error)
	UpdateActivity(ctx context.Context, a model.Activity) (model.Activity, error)
	CountActivitiesByOrganizerOpen(ctx context.Context, organizerID string) (int, error)
	CountActivitiesByStatus(ctx context.Context) (map[model.ActivityStatus]int, error)

	CreateSignup(ctx context.Context, s model.Signup) (model.Signup, error)
	GetSignup(ctx context.Context, id string) (model.Signup, error)
	GetSignupByActivityVolunteer(ctx context.Context, activityID, volunteerID string) (model.Signup, error)
	ListSignupsByActivity(ctx context.Context, activityID string) ([]model.Signup, error)
	ListSignupsByVolunteer(ctx context.Context, volunteerID string) ([]model.Signup, error)
	UpdateSignup(ctx context.Context, s model.Signup) (model.Signup, error)
	CountApprovedByActivity(ctx context.Context, activityID string) (int, error)
	CountWaitlistByActivity(ctx context.Context, activityID string) (int, error)
	NextWaitlistSeq(ctx context.Context, activityID string) (int, error)
	ListApprovedOverlapping(ctx context.Context, volunteerID string, start, end time.Time, excludeActivityID string) ([]model.Signup, error)
	CountNoShowsSince(ctx context.Context, volunteerID string, since time.Time) (int, error)

	CreateCheckIn(ctx context.Context, c model.CheckIn) (model.CheckIn, error)
	GetCheckIn(ctx context.Context, id string) (model.CheckIn, error)
	GetCheckInByActivityVolunteer(ctx context.Context, activityID, volunteerID string) (model.CheckIn, error)
	ListCheckInsByActivity(ctx context.Context, activityID string) ([]model.CheckIn, error)
	UpdateCheckIn(ctx context.Context, c model.CheckIn) (model.CheckIn, error)
	CountCheckInsOnDay(ctx context.Context, day time.Time) (int, error)

	CreateHour(ctx context.Context, h model.HourRecord) (model.HourRecord, error)
	GetHour(ctx context.Context, id string) (model.HourRecord, error)
	GetOpenHour(ctx context.Context, activityID, volunteerID string) (model.HourRecord, error)
	ListHoursByVolunteer(ctx context.Context, volunteerID string) ([]model.HourRecord, error)
	ListHoursByActivity(ctx context.Context, activityID string) ([]model.HourRecord, error)
	ListPendingHours(ctx context.Context) ([]model.HourRecord, error)
	UpdateHour(ctx context.Context, h model.HourRecord) (model.HourRecord, error)

	CreateHourLedger(ctx context.Context, l model.HourLedger) (model.HourLedger, error)
	ListHourLedgers(ctx context.Context, userID string) ([]model.HourLedger, error)
	CreatePointLedger(ctx context.Context, l model.PointLedger) (model.PointLedger, error)
	ListPointLedgers(ctx context.Context, userID string) ([]model.PointLedger, error)

	CreateTeam(ctx context.Context, t model.Team) (model.Team, error)
	GetTeam(ctx context.Context, id string) (model.Team, error)
	ListTeams(ctx context.Context, ownerID string) ([]model.Team, error)
	UpdateTeam(ctx context.Context, t model.Team) (model.Team, error)
	CreateTeamMember(ctx context.Context, m model.TeamMember) (model.TeamMember, error)
	ListTeamMembers(ctx context.Context, teamID string) ([]model.TeamMember, error)
	GetTeamMember(ctx context.Context, teamID, userID string) (model.TeamMember, error)
	DeleteTeamMember(ctx context.Context, teamID, userID string) error

	CreateNotification(ctx context.Context, n model.Notification) (model.Notification, error)
	ListNotifications(ctx context.Context, userID string, unreadOnly bool) ([]model.Notification, error)
	GetNotification(ctx context.Context, id string) (model.Notification, error)
	UpdateNotification(ctx context.Context, n model.Notification) (model.Notification, error)
	MarkAllNotificationsRead(ctx context.Context, userID string, at time.Time) (int, error)
	CountUnreadNotifications(ctx context.Context, userID string) (int, error)

	CreateAudit(ctx context.Context, a model.AuditLog) (model.AuditLog, error)
	ListAudits(ctx context.Context, targetType, targetID string) ([]model.AuditLog, error)

	CreateCertificate(ctx context.Context, c model.Certificate) (model.Certificate, error)
	GetCertificateByCode(ctx context.Context, code string) (model.Certificate, error)
	ListCertificatesByUser(ctx context.Context, userID string) ([]model.Certificate, error)
	HasCertificateTier(ctx context.Context, userID string, tier model.CertificateTier) (bool, error)

	CreateFeedback(ctx context.Context, f model.Feedback) (model.Feedback, error)
	GetFeedback(ctx context.Context, activityID, volunteerID string) (model.Feedback, error)
	ListFeedbackByActivity(ctx context.Context, activityID string) ([]model.Feedback, error)

	ApplyHoursAndPoints(ctx context.Context, hour model.HourRecord, user model.User, hl model.HourLedger, pl model.PointLedger) (model.HourRecord, model.User, error)
	ApplyPoints(ctx context.Context, user model.User, pl model.PointLedger) (model.User, error)
	RecountActivityCounters(ctx context.Context, activityID string) (model.Activity, error)
}
