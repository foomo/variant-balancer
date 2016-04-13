package usersessions

import (
	"log"
	"time"
)

// GCInterval set the interval for session garbage collection
var GCInterval = time.Second * 180

// @param maxAge: maximum Age of a SessionID. e.g: maxAge = 60s -> every SessionID without a visit in the 60s will be removed.
// @param minViews: minimum amount of pageviews, if not satisfied sessionID will be removed.
func (us *Sessions) collectGarbage(maxAge int64, minViews int) {
	log.Println("[DEBUG]: --------------------------------------------------", us.config.Id)
	log.Println("[DEBUG]: SessionID Garbage Collection Routine started!")
	log.Println("[DEBUG]: active sessionCookieNames:", us.sessionCookieNames)
	lowerBound := time.Now().Unix() - maxAge
	for cookieName, sessions := range us.UserSessions {
		for sessionId, session := range sessions {
			if session.Pageviews < int64(minViews) || session.LastVisit < lowerBound {
				us.sessionDeleteChannel <- &variantSessionPing{
					CookieName: cookieName,
					SessionId:  sessionId,
					VariantId:  session.VariantId,
				}
			}
		}
	}
}

func (us *Sessions) gcRoutine() {
	for {
		time.Sleep(GCInterval)
		if len(us.UserSessions) != 0 {
			// there are sessions
			us.collectGarbage(us.SessionTimeout, 2)
		}
		if us.Active == false {
			log.Println("user sessions expired", us.config.Id)
			return
		}
	}
}
