package model

import "time"

type ActivityStatus string

const (
	ActivityDraft               ActivityStatus = "draft"
	ActivityPublished           ActivityStatus = "published"
	ActivityRegistrationClosed  ActivityStatus = "registration_closed"
	ActivityInProgress          ActivityStatus = "in_progress"
	ActivityCompleted           ActivityStatus = "completed"
	ActivityCancelled           ActivityStatus = "cancelled"
)

func (s ActivityStatus) IsTerminal() bool {
	return s == ActivityCompleted || s == ActivityCancelled
}

func (s ActivityStatus) AllowsSignup() bool {
	return s == ActivityPublished
}

func (s ActivityStatus) AllowsManageRoster() bool {
	return s == ActivityPublished || s == ActivityRegistrationClosed || s == ActivityInProgress
}

func (s ActivityStatus) AllowsCheckIn() bool {
	return s == ActivityPublished || s == ActivityRegistrationClosed || s == ActivityInProgress
}

type Activity struct {
	ID                 string         `json:"id"`
	OrganizerID        string         `json:"organizer_id"`
	Title              string         `json:"title"`
	Content            string         `json:"content"`
	Category           CategoryID     `json:"category"`
	Location           string         `json:"location"`
	ContactName        string         `json:"contact_name"`
	ContactPhone       string         `json:"contact_phone"`
	Capacity           int            `json:"capacity"`
	WaitlistEnabled    bool           `json:"waitlist_enabled"`
	WaitlistLimit      int            `json:"waitlist_limit"`
	NeedApproval       bool           `json:"need_approval"`
	SignupOpenAt       time.Time      `json:"signup_open_at"`
	SignupCloseAt      time.Time      `json:"signup_close_at"`
	StartAt            time.Time      `json:"start_at"`
	EndAt              time.Time      `json:"end_at"`
	CheckInOpenBefore  int            `json:"check_in_open_before"`
	CheckOutGrace      int            `json:"check_out_grace"`
	PlannedMinutes     int            `json:"planned_minutes"`
	MinAge             int            `json:"min_age"`
	RequiredSkills     []string       `json:"required_skills"`
	TeamID             string         `json:"team_id,omitempty"`
	Status             ActivityStatus `json:"status"`
	CheckInCode        string         `json:"check_in_code"`
	ApprovedCount      int            `json:"approved_count"`
	WaitlistCount      int            `json:"waitlist_count"`
	CancelReason       string         `json:"cancel_reason,omitempty"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	PublishedAt        *time.Time     `json:"published_at,omitempty"`
	CompletedAt        *time.Time     `json:"completed_at,omitempty"`
}

func (a Activity) Public(viewer User) PublicActivity {
	p := PublicActivity{
		ID:                a.ID,
		OrganizerID:       a.OrganizerID,
		Title:             a.Title,
		Content:           a.Content,
		Category:          a.Category,
		CategoryName:      CategoryName(a.Category),
		Location:          a.Location,
		ContactName:       a.ContactName,
		Capacity:          a.Capacity,
		WaitlistEnabled:   a.WaitlistEnabled,
		WaitlistLimit:     a.WaitlistLimit,
		NeedApproval:      a.NeedApproval,
		SignupOpenAt:      a.SignupOpenAt,
		SignupCloseAt:     a.SignupCloseAt,
		StartAt:           a.StartAt,
		EndAt:             a.EndAt,
		CheckInOpenBefore: a.CheckInOpenBefore,
		CheckOutGrace:     a.CheckOutGrace,
		PlannedMinutes:    a.PlannedMinutes,
		MinAge:            a.MinAge,
		RequiredSkills:    append([]string(nil), a.RequiredSkills...),
		TeamID:            a.TeamID,
		Status:            a.Status,
		ApprovedCount:     a.ApprovedCount,
		WaitlistCount:     a.WaitlistCount,
		CreatedAt:         a.CreatedAt,
		UpdatedAt:         a.UpdatedAt,
		PublishedAt:       a.PublishedAt,
		CompletedAt:       a.CompletedAt,
	}
	if viewer.ID == a.OrganizerID || viewer.IsAdmin() {
		p.ContactPhone = a.ContactPhone
		p.CheckInCode = a.CheckInCode
	}
	return p
}

type PublicActivity struct {
	ID                string         `json:"id"`
	OrganizerID       string         `json:"organizer_id"`
	Title             string         `json:"title"`
	Content           string         `json:"content"`
	Category          CategoryID     `json:"category"`
	CategoryName      string         `json:"category_name"`
	Location          string         `json:"location"`
	ContactName       string         `json:"contact_name"`
	ContactPhone      string         `json:"contact_phone,omitempty"`
	Capacity          int            `json:"capacity"`
	WaitlistEnabled   bool           `json:"waitlist_enabled"`
	WaitlistLimit     int            `json:"waitlist_limit"`
	NeedApproval      bool           `json:"need_approval"`
	SignupOpenAt      time.Time      `json:"signup_open_at"`
	SignupCloseAt     time.Time      `json:"signup_close_at"`
	StartAt           time.Time      `json:"start_at"`
	EndAt             time.Time      `json:"end_at"`
	CheckInOpenBefore int            `json:"check_in_open_before"`
	CheckOutGrace     int            `json:"check_out_grace"`
	PlannedMinutes    int            `json:"planned_minutes"`
	MinAge            int            `json:"min_age"`
	RequiredSkills    []string       `json:"required_skills"`
	TeamID            string         `json:"team_id,omitempty"`
	Status            ActivityStatus `json:"status"`
	CheckInCode       string         `json:"check_in_code,omitempty"`
	ApprovedCount     int            `json:"approved_count"`
	WaitlistCount     int            `json:"waitlist_count"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	PublishedAt       *time.Time     `json:"published_at,omitempty"`
	CompletedAt       *time.Time     `json:"completed_at,omitempty"`
}

type ActivityFilter struct {
	Query       string
	Category    CategoryID
	Status      ActivityStatus
	OrganizerID string
	TeamID      string
	From        time.Time
	To          time.Time
	IncludeDraft bool
}
