package model

import "time"

type RegisterRequest struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
	Phone       string `json:"phone"`
	Age         int    `json:"age"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type UpdateProfileRequest struct {
	DisplayName string   `json:"display_name"`
	Phone       string   `json:"phone"`
	Bio         string   `json:"bio"`
	Age         int      `json:"age"`
	Skills      []string `json:"skills"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

type CreateUserRequest struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
	Phone       string `json:"phone"`
}

type CreateActivityRequest struct {
	Title             string     `json:"title"`
	Content           string     `json:"content"`
	Category          CategoryID `json:"category"`
	Location          string     `json:"location"`
	ContactName       string     `json:"contact_name"`
	ContactPhone      string     `json:"contact_phone"`
	Capacity          int        `json:"capacity"`
	WaitlistEnabled   bool       `json:"waitlist_enabled"`
	WaitlistLimit     int        `json:"waitlist_limit"`
	NeedApproval      bool       `json:"need_approval"`
	SignupOpenAt      time.Time  `json:"signup_open_at"`
	SignupCloseAt     time.Time  `json:"signup_close_at"`
	StartAt           time.Time  `json:"start_at"`
	EndAt             time.Time  `json:"end_at"`
	CheckInOpenBefore int        `json:"check_in_open_before"`
	CheckOutGrace     int        `json:"check_out_grace"`
	PlannedMinutes    int        `json:"planned_minutes"`
	MinAge            int        `json:"min_age"`
	RequiredSkills    []string   `json:"required_skills"`
	TeamID            string     `json:"team_id"`
	Publish           bool       `json:"publish"`
}

type SignupRequest struct {
	Remark string `json:"remark"`
}

type RejectRequest struct {
	Reason string `json:"reason"`
}

type CancelRequest struct {
	Reason string `json:"reason"`
}

type CheckInRequest struct {
	Code string `json:"code"`
}

type ProxyCheckInRequest struct {
	VolunteerID string     `json:"volunteer_id"`
	CheckInAt   *time.Time `json:"check_in_at"`
	Note        string     `json:"note"`
}

type SubmitHoursRequest struct {
	ActivityID   string `json:"activity_id"`
	VolunteerID  string `json:"volunteer_id"`
	WorkMinutes  int    `json:"work_minutes"`
	BreakMinutes int    `json:"break_minutes"`
	Note         string `json:"note"`
}

type ReviewHoursRequest struct {
	BreakMinutes int    `json:"break_minutes"`
	Note         string `json:"note"`
}

type CorrectHoursRequest struct {
	RequestedMinutes int    `json:"requested_minutes"`
	Note             string `json:"note"`
}

type FeedbackRequest struct {
	Score   int    `json:"score"`
	Comment string `json:"comment"`
}

type CreateTeamRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type InviteMemberRequest struct {
	Username string `json:"username"`
}
