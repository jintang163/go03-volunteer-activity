package hoursutil

import (
	"time"

	"go03-volunteer-activity/internal/policy"
)

func RawMinutes(checkIn, checkOut time.Time) int {
	if checkIn.IsZero() || checkOut.IsZero() {
		return 0
	}
	if !checkOut.After(checkIn) {
		return 0
	}
	d := checkOut.Sub(checkIn)
	mins := int(d / time.Minute)
	if mins < 0 {
		return 0
	}
	return mins
}

func WorkMinutes(checkIn, checkOut time.Time, breakMinutes, plannedMinutes int) int {
	raw := RawMinutes(checkIn, checkOut)
	return CapMinutes(raw, breakMinutes, plannedMinutes)
}

func CapMinutes(raw, breakMinutes, plannedMinutes int) int {
	if breakMinutes < 0 {
		breakMinutes = 0
	}
	if breakMinutes > policy.MaxBreakMinutes {
		breakMinutes = policy.MaxBreakMinutes
	}
	work := raw - breakMinutes
	if work < 0 {
		work = 0
	}
	if plannedMinutes > 0 && work > plannedMinutes {
		work = plannedMinutes
	}
	return work
}

func InCheckInWindow(now, start, end time.Time, openBefore, grace int) bool {
	open := start.Add(-time.Duration(openBefore) * time.Minute)
	closeAt := end.Add(time.Duration(grace) * time.Minute)
	if now.Before(open) {
		return false
	}
	if now.After(closeAt) {
		return false
	}
	return true
}

func InCheckOutWindow(now, start, end time.Time, grace int) bool {
	if now.Before(start) {
		return false
	}
	closeAt := end.Add(time.Duration(grace) * time.Minute)
	return !now.After(closeAt)
}

func NoShowDeadline(end time.Time, grace int) time.Time {
	return end.Add(time.Duration(grace) * time.Minute)
}
