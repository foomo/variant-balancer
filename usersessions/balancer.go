package usersessions

import (
	"time"
)

func (us *Sessions) getRandomVariant() *Variant {
	// initialize data
	stats := make(map[string]int64)
	sessionTimeoutTime := time.Now().Unix() - us.SessionTimeout
	for _, variant := range us.Variants {
		stats[variant.Id] = int64(0)
	}

	// count active sessions
	totalActiveSessionCount := int64(0)
	for _, userSession := range us.UserSessions {
		if userSession.Pageviews > 1 && userSession.LastVisit > sessionTimeoutTime {
			stats[userSession.VariantId]++
			totalActiveSessionCount++
		}
	}

	var v *Variant = nil
	distances := make(map[string]float64)
	// determine distance of desired share to actual share
	for variantId, activeSessions := range stats {
		variant := us.Variants[variantId]
		if totalActiveSessionCount > 0 {
			distances[variantId] = variant.Share - float64(activeSessions)/float64(totalActiveSessionCount)
			//log.Println("::::::::::::::::::::::::::::::::::::::", variantId, distances[variantId])
			if distances[variantId] < 0 {
				distances[variantId] = 0
			}
		} else {
			distances[variantId] = 0
		}
		//log.Println("variant distance ---------------------------------------------------------", variantId, distances[variantId])
		if v == nil || distances[variantId] > distances[v.Id] {
			//            if v == nil {
			//                log.Println("defaulting to", variant.Id)
			//            } else {
			//                log.Println("selecting", variantId, distances[variantId] > distances[v.Id])
			//            }

			v = variant
		}
	}
	return v
}

func (s *Sessions) getVariantForUserSessionId(sessionId string) *Variant {
	session, ok := s.UserSessions[sessionId]
	if ok {
		// log.Println("getVariantForUserSessionId: found user session", session)
		variant, ok := s.Variants[session.VariantId]
		if ok {
			return variant
		} else {
			//log.Println("getVariantForUserSessionId: there is no variant for id", session.VariantId)
		}
	}
	//log.Println("no variant for sessionId", sessionId, len(sessionId), session)
	return nil
}
