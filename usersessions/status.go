package usersessions

import (
	"github.com/foomo/variant-balancer/config"
	"strconv"
	"time"
)

type SessionStats struct {
	Sessions       int64
	ActiveSessions int64
	Pageviews      int64
	LastVisit      string
}

type SessionsStatus struct {
	Config       *config.Config
	Active       bool
	Stats        *SessionStats
	VariantStats map[string]*SessionStats
}

func (us *Sessions) GetStatus() *SessionsStatus {
	variantStats := make(map[string]*SessionStats)
	for _, variant := range us.Variants {
		variantStats[variant.Id] = us.getStatsForVariant(variant)
	}
	return &SessionsStatus{
		Active:       us.Active,
		Config:       us.config,
		Stats:        us.getStatsForVariant(nil),
		VariantStats: variantStats,
	}
}

func (us *Sessions) getStatsForVariant(variant *Variant) *SessionStats {
	views := int64(0)
	lastVisitTimestamp := int64(0)
	activeSessions := int64(0)
	sessionCount := int64(0)
	now := time.Now().Unix()
	for _, s := range us.UserSessions {
		if variant == nil || s.VariantId == variant.Id {
			views += s.Pageviews
			if s.LastVisit > lastVisitTimestamp {
				lastVisitTimestamp = s.LastVisit
			}
			sessionCount++
			if s.Pageviews > 1 && now-s.LastVisit < us.SessionTimeout {
				activeSessions++
			}
		}
	}
	ts := "---"
	if lastVisitTimestamp > 0 {
		t, err := time.ParseDuration(strconv.Itoa(int(now-lastVisitTimestamp)) + "s")
		if err == nil {
			ts = t.String()
		}
	}
	return &SessionStats{
		Sessions:       sessionCount,
		ActiveSessions: activeSessions,
		Pageviews:      views,
		LastVisit:      ts,
	}
}
