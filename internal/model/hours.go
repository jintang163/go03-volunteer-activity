package model

import "time"

type HourSource string

const (
	HourSourceAuto   HourSource = "auto"
	HourSourceManual HourSource = "manual"
)

type HourStatus string

const (
	HourDraft     HourStatus = "draft"
	HourPending   HourStatus = "pending"
	HourApproved  HourStatus = "approved"
	HourRejected  HourStatus = "rejected"
	HourCorrected HourStatus = "corrected"
)

type HourRecord struct {
	ID             string     `json:"id"`
	ActivityID     string     `json:"activity_id"`
	VolunteerID    string     `json:"volunteer_id"`
	SignupID       string     `json:"signup_id"`
	CheckInID      string     `json:"check_in_id,omitempty"`
	Source         HourSource `json:"source"`
	Status         HourStatus `json:"status"`
	RawMinutes     int        `json:"raw_minutes"`
	BreakMinutes   int        `json:"break_minutes"`
	WorkMinutes    int        `json:"work_minutes"`
	Note           string     `json:"note"`
	RejectReason   string     `json:"reject_reason,omitempty"`
	CorrectionNote string     `json:"correction_note,omitempty"`
	RequestedMins  int        `json:"requested_minutes,omitempty"`
	ReviewerID     string     `json:"reviewer_id,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	ApprovedAt     *time.Time `json:"approved_at,omitempty"`
}

type HourLedger struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	HourID      string    `json:"hour_id"`
	ActivityID  string    `json:"activity_id"`
	DeltaMin    int       `json:"delta_minutes"`
	BalanceMin  int       `json:"balance_minutes"`
	Note        string    `json:"note"`
	CreatedAt   time.Time `json:"created_at"`
}

type PointReason string

const (
	PointHours     PointReason = "hours"
	PointNoShow    PointReason = "no_show"
	PointLateCancel PointReason = "late_cancel"
	PointAdjust    PointReason = "adjust"
)

type PointLedger struct {
	ID         string      `json:"id"`
	UserID     string      `json:"user_id"`
	Delta      int         `json:"delta"`
	Balance    int         `json:"balance"`
	Reason     PointReason `json:"reason"`
	RelatedID  string      `json:"related_id"`
	Note       string      `json:"note"`
	CreatedAt  time.Time   `json:"created_at"`
}

type CertificateTier string

const (
	CertBronze CertificateTier = "bronze"
	CertSilver CertificateTier = "silver"
	CertGold   CertificateTier = "gold"
)

type Certificate struct {
	ID          string          `json:"id"`
	UserID      string          `json:"user_id"`
	Tier        CertificateTier `json:"tier"`
	Code        string          `json:"code"`
	HoursAtIssue int            `json:"hours_at_issue"`
	IssuedAt    time.Time       `json:"issued_at"`
	IssuerID    string          `json:"issuer_id"`
}

type Feedback struct {
	ID          string    `json:"id"`
	ActivityID  string    `json:"activity_id"`
	VolunteerID string    `json:"volunteer_id"`
	Score       int       `json:"score"`
	Comment     string    `json:"comment"`
	CreatedAt   time.Time `json:"created_at"`
}
