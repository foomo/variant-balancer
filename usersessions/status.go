package usersessions

import (
	"strconv"
	"time"

	"github.com/foomo/variant-balancer/config"
)

type SessionStats struct {
	Sessions       int64   `json:"sessions"`
	ActiveSessions int64   `json:"activeSessions"`
	ActiveShare    float64 `json:"activeShare"`
	Share          float64 `json:"share"`
	Pageviews      int64   `json:"pageViews"`
	LastVisit      string  `json:"lastVisit"`
}

type SessionsStatus struct {
	Config       *config.Config           `json:"config"`
	Active       bool                     `json:"active"`
	Stats        *SessionStats            `json:"stats"`
	VariantStats map[string]*SessionStats `json:"variantStats"`
}

func (us *Sessions) GetStatus() *SessionsStatus {
	return &SessionsStatus{
		Active:       us.Active,
		Config:       us.config,
		Stats:        getStatsForVariant(nil, us.userSessions, us.SessionTimeout),
		VariantStats: getVariantStats(us.Variants, us.userSessions, us.SessionTimeout),
	}
}

func getVariantStats(variants map[string]*Variant, userSessions *userSessions, sessionTimeout int64) map[string]*SessionStats {
	variantStats := map[string]*SessionStats{}
	for _, variant := range variants {
		variantStats[variant.Id] = getStatsForVariant(variant, userSessions, sessionTimeout)
	}
	return variantStats
}

func getStatsForVariant(variant *Variant, userSessions *userSessions, sessionTimeout int64) *SessionStats {
	views := int64(0)
	lastVisitTimestamp := int64(0)
	activeSessions := int64(0)
	activeSessionsTotal := int64(0)
	sessionCount := int64(0)

	now := time.Now().Unix()
	userSessions.RLock()
	for _, sessions := range userSessions.m {
		for _, s := range sessions {
			sessionIsActive := s.Pageviews > 1 && now-s.LastVisit < sessionTimeout
			if sessionIsActive {
				activeSessionsTotal++
			}
			if variant == nil || s.VariantId == variant.Id {
				views += s.Pageviews
				if s.LastVisit > lastVisitTimestamp {
					lastVisitTimestamp = s.LastVisit
				}
				sessionCount++
				if sessionIsActive {
					activeSessions++
				}
			}

		}
	}
	userSessions.RUnlock()
	ts := "---"
	if lastVisitTimestamp > 0 {
		t, err := time.ParseDuration(strconv.Itoa(int(now-lastVisitTimestamp)) + "s")
		if err == nil {
			ts = t.String()
		}
	}
	activeShare := float64(0)
	if activeSessionsTotal > 0 {
		activeShare = float64(activeSessions) / float64(activeSessionsTotal)
	}
	var share float64
	if variant != nil {
		share = variant.Share
	}
	return &SessionStats{
		Sessions:       sessionCount,
		ActiveSessions: activeSessions,
		Share:          share,
		ActiveShare:    activeShare,
		Pageviews:      views,
		LastVisit:      ts,
	}
}
