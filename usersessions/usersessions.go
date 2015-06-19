package usersessions

import (
	"github.com/foomo/variant-balancer/config"
	//"log"
	"net/http"
	"time"
)

type variantSessionPing struct {
	SessionId  string
	VariantId  string
	CookieName string
}

type Sessions struct {
	// will i accept new client or am I about to gracefully shutdown
	SessionTimeout       int64
	Active               bool
	UserSessions         map[string]map[string]*UserSession
	Variants             map[string]*Variant
	sessionPingChannel   chan *variantSessionPing
	sessionDeleteChannel chan *variantSessionPing
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
		UserSessions:         make(map[string]map[string]*UserSession),
		Variants:             variants,
		sessionPingChannel:   make(chan *variantSessionPing),
		sessionDeleteChannel: make(chan *variantSessionPing),
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
	//go us.gcRoutine()

	return us
}

func (us *Sessions) sessionPingRoutine() {
	for {
		select {
		case sessionPingOfDeath := <-us.sessionDeleteChannel:
			delete(us.UserSessions[sessionPingOfDeath.CookieName], sessionPingOfDeath.SessionId)
		case sessionPing := <-us.sessionPingChannel:
			if len(sessionPing.SessionId) > 0 {
				_, ok := us.UserSessions[sessionPing.CookieName]
				if !ok {
					us.UserSessions[sessionPing.CookieName] = make(map[string]*UserSession)
				}
				session, ok := us.UserSessions[sessionPing.CookieName][sessionPing.SessionId]
				if !ok {
					//log.Println("[DEBUG]: creating new Session!")
					session = NewUserSession(sessionPing)
					us.UserSessions[sessionPing.CookieName][sessionPing.SessionId] = session
				}
				session.LastVisit = time.Now().Unix()
				session.Pageviews++
			} else {
				//log.Println("[DEBUG]: sessionPing.SessionId is empty!")
			}
		}
	}
}

func (s *Sessions) extractSessionId(incomingRequest *http.Request) (sessionId string, cookieName string) {
	//log.Println("extractSessionId", s.sessionCookieNames, incomingRequest.Cookies())
	for _, cookieName = range s.sessionCookieNames {
		cookie, err := incomingRequest.Cookie(cookieName)
		if err == nil && cookie != nil && len(cookie.Value) > 0 {
			//log.Println("found", cookieName, cookie)
			return cookie.Value, cookieName
		}
	}
	//log.Println("cookie not found")
	return "", ""
}

// get an existing user variant
func (us *Sessions) GetExistingUserVariant(incomingRequest *http.Request) *Variant {
	sessionId, cookieName := us.extractSessionId(incomingRequest)
	if len(sessionId) > 0 {
		return us.getVariantForUserSessionId(sessionId, cookieName)
	}
	return nil
}

func (us *Sessions) GetBalancedRandomVariant() *Variant {
	variantId := getBalancedRandomVariantId(getVariantStats(us.Variants, us.UserSessions, us.SessionTimeout))
	variant, _ := us.Variants[variantId]
	return variant
}

func (us *Sessions) serveVariant(variant *Variant, w http.ResponseWriter, incomingRequest *http.Request, cacheId string) (sessionId string, err error) {
	sessionId, cookieName, err := variant.Serve(w, incomingRequest, cacheId)
	if err == nil && len(sessionId) > 0 {
		us.sessionPingChannel <- &variantSessionPing{
			SessionId:  sessionId,
			VariantId:  variant.Id,
			CookieName: cookieName,
		}
	}
	return

}
func (us *Sessions) ServeVariant(variant *Variant, w http.ResponseWriter, incomingRequest *http.Request, cacheId string) (err error) {
	_, err = us.serveVariant(variant, w, incomingRequest, cacheId)
	return err
}
