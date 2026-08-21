package model

import "errors"

var (
	ErrNotFound           = errors.New("resource not found")
	ErrAlreadyExists      = errors.New("resource already exists")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden")
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrValidation         = errors.New("validation error")
	ErrConflict           = errors.New("conflict")
	ErrInternal           = errors.New("internal error")
	ErrAccountFrozen      = errors.New("account is frozen")
	ErrAccountBanned      = errors.New("account is banned")

	ErrInvalidUsername    = errors.New("invalid username: 3-32 letters, digits or underscore")
	ErrInvalidPassword    = errors.New("invalid password: 6-64 characters")
	ErrInvalidRole        = errors.New("invalid role: must be admin, organizer or volunteer")
	ErrInvalidDisplayName = errors.New("invalid display name: 1-32 characters")
	ErrInvalidPhone       = errors.New("invalid phone: 0-20 characters")
	ErrInvalidBio         = errors.New("invalid bio: max 200 characters")
	ErrInvalidTitle       = errors.New("invalid title: 1-80 characters")
	ErrInvalidContent     = errors.New("invalid content: 1-4000 characters")
	ErrInvalidCategory    = errors.New("invalid category")
	ErrInvalidStatus      = errors.New("invalid status")
	ErrInvalidUserStatus  = errors.New("invalid user status")
	ErrInvalidTimeWindow  = errors.New("invalid time window: end must be after start")
	ErrInvalidCapacity    = errors.New("invalid capacity: must be 1-500")
	ErrInvalidMinutes     = errors.New("invalid minutes")
	ErrInvalidScore       = errors.New("invalid score: 1-5")
	ErrInvalidCheckInCode = errors.New("invalid check-in code")
	ErrInvalidAge         = errors.New("invalid age")

	ErrNotOrganizer           = errors.New("only the activity organizer can perform this action")
	ErrNotVolunteer           = errors.New("only volunteers can perform this action")
	ErrCannotSignupOwn        = errors.New("cannot sign up for your own activity")
	ErrAlreadySignedUp        = errors.New("already signed up for this activity")
	ErrActivityNotOpen        = errors.New("activity is not open for signup")
	ErrSignupWindowClosed     = errors.New("signup window is closed")
	ErrSignupWindowNotOpen    = errors.New("signup window is not open yet")
	ErrCapacityFull           = errors.New("activity is full and waitlist is unavailable")
	ErrWaitlistFull           = errors.New("waitlist is full")
	ErrScheduleConflict       = errors.New("schedule conflicts with another approved activity")
	ErrTooManyNoShows         = errors.New("too many no-shows in 90 days")
	ErrSkillMismatch          = errors.New("volunteer does not meet required skills")
	ErrAgeRequirement         = errors.New("volunteer does not meet minimum age")
	ErrTeamOnly               = errors.New("only team members can sign up")
	ErrNotApprovedSignup      = errors.New("signup is not approved")
	ErrInvalidActivityStatus  = errors.New("activity status does not allow this action")
	ErrAlreadyCheckedIn       = errors.New("already checked in")
	ErrNotCheckedIn           = errors.New("not checked in")
	ErrAlreadyCheckedOut      = errors.New("already checked out")
	ErrCheckInWindowClosed    = errors.New("check-in window is closed")
	ErrCheckOutWindowClosed   = errors.New("check-out window is closed")
	ErrWrongCheckInCode       = errors.New("wrong check-in code")
	ErrHoursAlreadyExists     = errors.New("an open or approved hour record already exists")
	ErrHoursNotPending        = errors.New("hour record is not pending review")
	ErrNoCheckoutForAutoHours = errors.New("cannot auto-calculate hours without check-out")
	ErrAlreadyFeedback        = errors.New("already submitted feedback")
	ErrActivityNotCompleted   = errors.New("activity is not completed")
	ErrAlreadyCertTier        = errors.New("certificate tier already issued")
	ErrNotTeamOwner           = errors.New("only the team owner can perform this action")
	ErrAlreadyTeamMember      = errors.New("already a team member")
	ErrNotTeamMember          = errors.New("not a team member")
	ErrCannotSelfRegisterOrg  = errors.New("organizer accounts must be created by admin")
)

func IsNotFound(err error) bool { return errors.Is(err, ErrNotFound) }

func IsAlreadyExists(err error) bool { return errors.Is(err, ErrAlreadyExists) }

func IsUnauthorized(err error) bool { return errors.Is(err, ErrUnauthorized) }

func IsInvalidCredentials(err error) bool { return errors.Is(err, ErrInvalidCredentials) }

func IsForbidden(err error) bool {
	return errors.Is(err, ErrForbidden) ||
		errors.Is(err, ErrAccountFrozen) ||
		errors.Is(err, ErrAccountBanned) ||
		errors.Is(err, ErrNotOrganizer) ||
		errors.Is(err, ErrNotVolunteer) ||
		errors.Is(err, ErrNotTeamOwner) ||
		errors.Is(err, ErrCannotSelfRegisterOrg)
}

func IsConflict(err error) bool {
	return errors.Is(err, ErrConflict) ||
		errors.Is(err, ErrCannotSignupOwn) ||
		errors.Is(err, ErrAlreadySignedUp) ||
		errors.Is(err, ErrActivityNotOpen) ||
		errors.Is(err, ErrSignupWindowClosed) ||
		errors.Is(err, ErrSignupWindowNotOpen) ||
		errors.Is(err, ErrCapacityFull) ||
		errors.Is(err, ErrWaitlistFull) ||
		errors.Is(err, ErrScheduleConflict) ||
		errors.Is(err, ErrTooManyNoShows) ||
		errors.Is(err, ErrSkillMismatch) ||
		errors.Is(err, ErrAgeRequirement) ||
		errors.Is(err, ErrTeamOnly) ||
		errors.Is(err, ErrNotApprovedSignup) ||
		errors.Is(err, ErrInvalidActivityStatus) ||
		errors.Is(err, ErrAlreadyCheckedIn) ||
		errors.Is(err, ErrNotCheckedIn) ||
		errors.Is(err, ErrAlreadyCheckedOut) ||
		errors.Is(err, ErrCheckInWindowClosed) ||
		errors.Is(err, ErrCheckOutWindowClosed) ||
		errors.Is(err, ErrWrongCheckInCode) ||
		errors.Is(err, ErrHoursAlreadyExists) ||
		errors.Is(err, ErrHoursNotPending) ||
		errors.Is(err, ErrNoCheckoutForAutoHours) ||
		errors.Is(err, ErrAlreadyFeedback) ||
		errors.Is(err, ErrActivityNotCompleted) ||
		errors.Is(err, ErrAlreadyCertTier) ||
		errors.Is(err, ErrAlreadyTeamMember) ||
		errors.Is(err, ErrNotTeamMember)
}

func IsValidation(err error) bool {
	switch {
	case errors.Is(err, ErrValidation),
		errors.Is(err, ErrInvalidUsername),
		errors.Is(err, ErrInvalidPassword),
		errors.Is(err, ErrInvalidRole),
		errors.Is(err, ErrInvalidDisplayName),
		errors.Is(err, ErrInvalidPhone),
		errors.Is(err, ErrInvalidBio),
		errors.Is(err, ErrInvalidTitle),
		errors.Is(err, ErrInvalidContent),
		errors.Is(err, ErrInvalidCategory),
		errors.Is(err, ErrInvalidStatus),
		errors.Is(err, ErrInvalidUserStatus),
		errors.Is(err, ErrInvalidTimeWindow),
		errors.Is(err, ErrInvalidCapacity),
		errors.Is(err, ErrInvalidMinutes),
		errors.Is(err, ErrInvalidScore),
		errors.Is(err, ErrInvalidCheckInCode),
		errors.Is(err, ErrInvalidAge):
		return true
	}
	return false
}
