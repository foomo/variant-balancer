package usersessions

import (
	"github.com/foomo/variant-balancer/config"
	//"log"
	"net/http"
	"time"
)

type VariantSessionPing struct {
	SessionId string
	VariantId string
}

type Sessions struct {
	// will i accept new client or am I about to gracefully shutdown
	SessionTimeout       int64
	Active               bool
	UserSessions         map[string]*UserSession
	Variants             map[string]*Variant
	SessionPingChannel   chan *VariantSessionPing
	sessionDeleteChannel chan string
	sessionCookieNames   []string
	config               *config.Config
}

func NewSessions(c *config.Config) *Sessions {
	variants := make(map[string]*Variant)
	for _, variantConfig := range c.Variants {
		variants[variantConfig.Id] = NewVariant(variantConfig)
	}
	us := &Sessions{
		Active:               true,
		UserSessions:         make(map[string]*UserSession),
		Variants:             variants,
		SessionPingChannel:   make(chan *VariantSessionPing),
		sessionDeleteChannel: make(chan string),
		sessionCookieNames:   []string{},
		SessionTimeout:       int64(c.SessionTimeout),
		config:               c,
	}

	for _, variant := range us.Variants {
		for _, node := range variant.Proxy.Nodes {
			us.sessionCookieNames = append(us.sessionCookieNames, node.SessionCookieName)
		}
	}

	// ready to collect session data
	go us.sessionPingRoutine()

	// starting garbage collection routine
	go us.gcRoutine()

	return us
}

func (us *Sessions) sessionPingRoutine() {
	for {
		select {
		case sessionDeleteId := <-us.sessionDeleteChannel:
			delete(us.UserSessions, sessionDeleteId)
		case sessionPing := <-us.SessionPingChannel:
			if len(sessionPing.SessionId) > 0 {
				session, ok := us.UserSessions[sessionPing.SessionId]
				if !ok {
					//log.Println("[DEBUG]: creating new Session!")
					session = NewUserSession(sessionPing)
					us.UserSessions[sessionPing.SessionId] = session
				}
				session.LastVisit = time.Now().Unix()
				session.Pageviews++
			} else {
				//log.Println("[DEBUG]: sessionPing.SessionId is empty!")
			}
		}
	}
}

func (s *Sessions) extractSessionId(incomingRequest *http.Request) string {
	//log.Println("extractSessionId", s.sessionCookieNames, incomingRequest.Cookies())
	for _, cookieName := range s.sessionCookieNames {
		cookie, err := incomingRequest.Cookie(cookieName)
		if err == nil && cookie != nil && len(cookie.Value) > 0 {
			// log.Println("found", cookieName, cookie)
			return cookieName + cookie.Value
		} else {
			// log.Println("err for", cookieName, cookie, err)
		}
	}
	return ""
}

func (us *Sessions) getUserVariant(incomingRequest *http.Request) (v *Variant, sessionId string) {
	sessionId = us.extractSessionId(incomingRequest)
	// log.Println("getting user variant for sessionId", sessionId)
	if len(sessionId) > 0 {
		// there is a session cookie
		return us.getVariantForUserSessionId(sessionId), sessionId
	} else if us.Active {
		return us.getRandomVariant(), sessionId
	}
	return nil, sessionId
}

func (us *Sessions) serveVariant(variant *Variant, w http.ResponseWriter, incomingRequest *http.Request, cacheId string) (sessionId string, err error) {
	sessionId, err = variant.Serve(w, incomingRequest, cacheId)
	if err == nil && len(sessionId) > 0 {
		us.SessionPingChannel <- &VariantSessionPing{
			SessionId: sessionId,
			VariantId: variant.Id,
		}
	}
	return sessionId, err
}

func (us *Sessions) Serve(w http.ResponseWriter, incomingRequest *http.Request, cacheId string) (sessionId string, err error) {
	variant, _ := us.getUserVariant(incomingRequest)
	return us.serveVariant(variant, w, incomingRequest, cacheId)
}
