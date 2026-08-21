package model

import "time"

type SignupStatus string

const (
	SignupPending    SignupStatus = "pending"
	SignupApproved   SignupStatus = "approved"
	SignupWaitlisted SignupStatus = "waitlisted"
	SignupRejected   SignupStatus = "rejected"
	SignupCancelled  SignupStatus = "cancelled"
	SignupNoShow     SignupStatus = "no_show"
)

func (s SignupStatus) IsActive() bool {
	return s == SignupPending || s == SignupApproved || s == SignupWaitlisted
}

type Signup struct {
	ID            string       `json:"id"`
	ActivityID    string       `json:"activity_id"`
	VolunteerID   string       `json:"volunteer_id"`
	Status        SignupStatus `json:"status"`
	Remark        string       `json:"remark"`
	RejectReason  string       `json:"reject_reason,omitempty"`
	WaitlistSeq   int          `json:"waitlist_seq"`
	ApprovedAt    *time.Time   `json:"approved_at,omitempty"`
	CancelledAt   *time.Time   `json:"cancelled_at,omitempty"`
	CreatedAt     time.Time    `json:"created_at"`
	UpdatedAt     time.Time    `json:"updated_at"`
}

type PublicSignup struct {
	Signup
	Volunteer PublicUser `json:"volunteer"`
}

type CheckInMethod string

const (
	CheckInSelf      CheckInMethod = "self"
	CheckInOrganizer CheckInMethod = "organizer"
	CheckInCode      CheckInMethod = "code"
)

type CheckIn struct {
	ID          string        `json:"id"`
	ActivityID  string        `json:"activity_id"`
	VolunteerID string        `json:"volunteer_id"`
	SignupID    string        `json:"signup_id"`
	Method      CheckInMethod `json:"method"`
	CheckInAt   time.Time     `json:"check_in_at"`
	CheckOutAt  *time.Time    `json:"check_out_at,omitempty"`
	Note        string        `json:"note,omitempty"`
	ActorID     string        `json:"actor_id"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

func (c CheckIn) HasCheckedOut() bool {
	return c.CheckOutAt != nil && !c.CheckOutAt.IsZero()
}
