package usersessions

import (
	"github.com/foomo/variant-balancer/config"
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

// get an existing user variant
func (us *Sessions) GetExistingUserVariant(incomingRequest *http.Request) *Variant {
	sessionId := us.extractSessionId(incomingRequest)
	if len(sessionId) > 0 {
		return us.getVariantForUserSessionId(sessionId)
	}
	return nil
}

func (us *Sessions) GetBalancedRandomVariant() *Variant {
	variantId := getBalancedRandomVariantId(getVariantStats(us.Variants, us.UserSessions, us.SessionTimeout))
	variant, _ := us.Variants[variantId]
	return variant
}

func (us *Sessions) ServeVariant(variant *Variant, w http.ResponseWriter, incomingRequest *http.Request, cacheId string) (err error) {
	sessionId, err := variant.Serve(w, incomingRequest, cacheId)
	if err == nil && len(sessionId) > 0 {
		us.SessionPingChannel <- &VariantSessionPing{
			SessionId: sessionId,
			VariantId: variant.Id,
		}
	}
	return err
}
