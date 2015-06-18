package usersessions

import (
	"log"
	"time"
)

// @param maxAge: maximum Age of a SessionID. e.g: maxAge = 60s -> every SessionID without a visit in the 60s will be removed.
// @param minViews: minimum amount of pageviews, if not satisfied sessionID will be removed.
func (us *Sessions) collectGarbage(maxAge int64, minViews int) {
	log.Println("[DEBUG]: --------------------------------------------------")
	log.Println("[DEBUG]: SessionID Garbage Collection Routine started!")
	log.Println("[DEBUG]: active sessionCookieNames:", us.sessionCookieNames)

	log.Println("[DEBUG]: active sessions:")
	count := 0
	for _, session := range us.UserSessions {
		count++
		log.Println("[DEBUG]: #", count, "UserSession.Id:", session.Id, "UserSession.Pageviews:", session.Pageviews, "UserSession.Lastvisit:", session.LastVisit)
	}

	lowerBound := time.Now().Unix() - maxAge
	for key, session := range us.UserSessions {
		if session.Pageviews < int64(minViews) || session.LastVisit < lowerBound {
			us.sessionDeleteChannel <- key
			log.Println("[DEBUG]: deleted", key, "from map!")
		}
	}
}

func (us *Sessions) gcRoutine() {
	for {
		time.Sleep(time.Second)
		if len(us.UserSessions) != 0 {
			// there are sessions
			us.collectGarbage(us.SessionTimeout, 2)
		}
	}
}
