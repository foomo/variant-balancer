package usersessions

import "time"

// GCInterval set the interval for session garbage collection
var GCInterval = time.Second * 180

// @param maxAge: maximum Age of a SessionID. e.g: maxAge = 60s -> every SessionID without a visit in the 60s will be removed.
// @param minViews: minimum amount of pageviews, if not satisfied sessionID will be removed.
func (us *Sessions) collectGarbage(maxAge int64, minViews int) {
	lowerBound := time.Now().Unix() - maxAge
	pings := []*variantSessionPing{}
	us.userSessions.RLock()
	for cookieName, sessions := range us.userSessions.m {
		for sessionId, session := range sessions {
			if session.Pageviews < int64(minViews) || session.LastVisit < lowerBound {
				pings = append(pings, &variantSessionPing{
					CookieName: cookieName,
					SessionId:  sessionId,
					VariantId:  session.VariantId,
				})
			}
		}
	}
	us.userSessions.RUnlock()
	us.sessionDeleteChannel <- pings
}

func (us *Sessions) gcRoutine() {
	for {
		time.Sleep(GCInterval)
		if us.userSessions != nil && len(us.userSessions.m) != 0 {
			// there are sessions
			us.collectGarbage(us.SessionTimeout, 2)
		}
		if us.Active == false {
			return
		}
	}
}
