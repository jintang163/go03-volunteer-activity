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
	// ReserveActivityCompletion 在单次写锁内原子地完成「校验活动尚未处于终态 + 写入已完成状态」。
	// 它消除 Complete 此前「先 GetActivity 读状态、IsTerminal 检查、后 UpdateActivity 写入」两步之间的
	// TOCTOU 竞态：同一场已结束活动的两个并发完成请求都读到非终态而通过检查后，第二个进入写锁时
	// 重新读到活动已是 completed，即返回 ErrInvalidActivityStatus，从而保证活动状态只能迁移一次，
	// 且缺席扣分、通知等后续副作用只随胜出请求触发一次。
	ReserveActivityCompletion(ctx context.Context, a model.Activity) (model.Activity, error)
	CountActivitiesByOrganizerOpen(ctx context.Context, organizerID string) (int, error)
	CountActivitiesByStatus(ctx context.Context) (map[model.ActivityStatus]int, error)

	CreateSignup(ctx context.Context, s model.Signup) (model.Signup, error)
	// ReserveSignup 在单次写锁内原子地完成容量/候补/冲突/审批判定并写入报名记录，
	// 消除「先 CountApprovedByActivity 检查、后 CreateSignup 写入」之间的 TOCTOU 竞态：
	// 同一时刻最多只有一个并发请求能通过容量检查，其余在锁内重新计数时发现已满而失败。
	ReserveSignup(ctx context.Context, s model.Signup, act model.Activity) (model.Signup, error)
	GetSignup(ctx context.Context, id string) (model.Signup, error)
	GetSignupByActivityVolunteer(ctx context.Context, activityID, volunteerID string) (model.Signup, error)
	ListSignupsByActivity(ctx context.Context, activityID string) ([]model.Signup, error)
	ListSignupsByVolunteer(ctx context.Context, volunteerID string) ([]model.Signup, error)
	UpdateSignup(ctx context.Context, s model.Signup) (model.Signup, error)
	// ReserveSignupApproval 在单次写锁内原子地完成「校验报名仍待审批 + 容量是否已满 +
	// 时间是否冲突 + 写入已录取状态」。它消除 Approve 此前「先 CountApprovedByActivity 检查容量、
	// ListApprovedOverlapping 检查冲突、后 UpdateSignup 写入」两步之间的 TOCTOU 竞态：两个并发
	// 审批请求都读到 approved < capacity 而通过检查后，第二个进入写锁时重新计数发现已达上限即
	// 返回 ErrCapacityFull，从而保证最后一个名额只能被一条报名成功占用，且活动录取计数保持一致。
	ReserveSignupApproval(ctx context.Context, s model.Signup, act model.Activity) (model.Signup, error)
	// ReserveSignupCancellation 在单次写锁内原子地完成「校验报名仍处于有效状态 + 写入已取消状态」。
	// 它消除 Cancel 此前「先 GetSignup 读取状态、IsActive 检查、后 UpdateSignup 写入」两步之间的
	// TOCTOU 竞态：同一条已录取报名在活动开始后的两个并发取消请求都读到有效状态而通过检查后，
	// 第二个进入写锁时重新读到报名已是 cancelled，即返回 ErrConflict，从而保证同一报名的取消状态
	// 只能迁移一次，且迟到扣分、候补递补等后续副作用只随胜出请求触发一次。
	ReserveSignupCancellation(ctx context.Context, s model.Signup, act model.Activity) (model.Signup, error)
	CountApprovedByActivity(ctx context.Context, activityID string) (int, error)
	CountWaitlistByActivity(ctx context.Context, activityID string) (int, error)
	NextWaitlistSeq(ctx context.Context, activityID string) (int, error)
	ListApprovedOverlapping(ctx context.Context, volunteerID string, start, end time.Time, excludeActivityID string) ([]model.Signup, error)
	CountNoShowsSince(ctx context.Context, volunteerID string, since time.Time) (int, error)

	CreateCheckIn(ctx context.Context, c model.CheckIn) (model.CheckIn, error)
	// ReserveCheckIn 在单次写锁内原子地完成「检查同一活动同一志愿者是否已签到 + 写入记录」，
	// 消除「先 GetCheckInByActivityVolunteer 检查、后 CreateCheckIn 写入」之间的 TOCTOU 竞态：
	// 同一时刻最多只有一个并发请求能在锁内确认尚无签到记录并写入，其余进入锁内时已读到
	// 已存在的签到记录，即返回 ErrAlreadyCheckedIn，从而保证同一志愿者在同一活动只能签到一次。
	ReserveCheckIn(ctx context.Context, c model.CheckIn) (model.CheckIn, error)
	GetCheckIn(ctx context.Context, id string) (model.CheckIn, error)
	GetCheckInByActivityVolunteer(ctx context.Context, activityID, volunteerID string) (model.CheckIn, error)
	ListCheckInsByActivity(ctx context.Context, activityID string) ([]model.CheckIn, error)
	UpdateCheckIn(ctx context.Context, c model.CheckIn) (model.CheckIn, error)
	// ReserveCheckOut 在单次写锁内原子地完成「校验签到记录尚未签退 + 写入签退时间」，
	// 消除 CheckOut 此前「先 GetCheckInByActivityVolunteer 检查 HasCheckedOut、后 UpdateCheckIn
	// 写入」之间的 TOCTOU 竞态：同一志愿者的两个并发签退请求都读到未签退状态而通过检查后，
	// 第二个进入写锁时重新读到记录已带签退时间，即返回 ErrAlreadyCheckedOut，从而保证同一
	// 签到记录的签退状态只能迁移一次，且签退后的工时草拟只随胜出请求触发一次。
	ReserveCheckOut(ctx context.Context, c model.CheckIn) (model.CheckIn, error)
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
	// ReserveTeamMember 在单次写锁内原子地完成「检查同一团队同一用户是否已是成员 + 写入记录」，
	// 消除「先 GetTeamMember 检查、后 CreateTeamMember 写入」之间的 TOCTOU 竞态：
	// 同一时刻最多只有一个并发请求能在锁内确认尚无成员关系并写入，其余进入锁内时已读到
	// 已存在的成员记录，即返回 ErrAlreadyTeamMember，从而保证同一用户在同一团队只能建立一条成员关系。
	ReserveTeamMember(ctx context.Context, m model.TeamMember) (model.TeamMember, error)
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
	// ReserveFeedback 在单次写锁内原子地完成「检查同一活动同一志愿者是否已反馈 + 写入记录」，
	// 消除「先 GetFeedback 检查、后 CreateFeedback 写入」之间的 TOCTOU 竞态：
	// 同一时刻最多只有一个并发请求能在锁内确认尚无反馈记录并写入，其余进入锁内时已读到
	// 已存在的反馈记录，即返回 ErrAlreadyFeedback，从而保证同一志愿者在同一活动只能反馈一次。
	ReserveFeedback(ctx context.Context, f model.Feedback) (model.Feedback, error)

	// ApplyHourApproval 在单次写锁内原子地完成「校验工时仍在待审 + 入账工时与积分 +
	// 写两条流水」。它把 Approve 此前「先 GetHour 检查状态、后入账」两步间留下的
	// TOCTOU 竞态消除：并发审批都通过状态检查后，第二个进入锁内时看到工时已是
	// approved，即返回 ErrHoursNotPending，从而保证同一条工时只能成功入账一次。
	ApplyHourApproval(ctx context.Context, hour model.HourRecord, hl model.HourLedger, pl model.PointLedger) (model.HourRecord, model.User, error)
	ApplyPoints(ctx context.Context, user model.User, pl model.PointLedger) (model.User, error)
	RecountActivityCounters(ctx context.Context, activityID string) (model.Activity, error)
}
