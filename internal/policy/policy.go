package policy

import "time"

const (
	DefaultCheckInOpenBefore = 30
	DefaultCheckOutGrace     = 60
	DefaultWaitlistLimit     = 20
	MaxCapacity              = 500
	MinCapacity              = 1
	MaxOpenActivitiesPerOrg  = 20
	MaxApprovedConcurrent    = 5
	NoShowWindowDays         = 90
	DefaultNoShowLimit       = 3
	NoShowPointPenalty       = 15
	LateCancelPointPenalty   = 8
	PointsPerSixMinutes      = 1
	BronzeMinutes            = 20 * 60
	SilverMinutes            = 50 * 60
	GoldMinutes              = 100 * 60
	MaxBreakMinutes          = 240
	MaxPlannedMinutes        = 24 * 60
	MinPlannedMinutes        = 30
	ProxyCheckInMaxLate      = 48 * time.Hour
	UsernameMin              = 3
	UsernameMax              = 32
	PasswordMin              = 6
	PasswordMax              = 64
	DisplayNameMax           = 32
	TitleMax                 = 80
	ContentMax               = 4000
	RemarkMax                = 200
)

func PointsForMinutes(minutes int) int {
	if minutes <= 0 {
		return 0
	}
	return minutes / 6 * PointsPerSixMinutes
}

func CertificateTier(totalMinutes int) string {
	switch {
	case totalMinutes >= GoldMinutes:
		return "gold"
	case totalMinutes >= SilverMinutes:
		return "silver"
	case totalMinutes >= BronzeMinutes:
		return "bronze"
	default:
		return ""
	}
}

func NextCertificateTier(totalMinutes int) (tier string, remain int) {
	switch {
	case totalMinutes < BronzeMinutes:
		return "bronze", BronzeMinutes - totalMinutes
	case totalMinutes < SilverMinutes:
		return "silver", SilverMinutes - totalMinutes
	case totalMinutes < GoldMinutes:
		return "gold", GoldMinutes - totalMinutes
	default:
		return "", 0
	}
}

func Overlap(aStart, aEnd, bStart, bEnd time.Time) bool {
	if aStart.IsZero() || aEnd.IsZero() || bStart.IsZero() || bEnd.IsZero() {
		return false
	}
	return aStart.Before(bEnd) && bStart.Before(aEnd)
}
