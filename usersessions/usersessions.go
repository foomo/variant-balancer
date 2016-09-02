package usersessions

import (
	"sync"

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

type userSessions struct {
	sync.RWMutex
	m map[string]map[string]*UserSession
}

type Sessions struct {
	// will i accept new client or am I about to gracefully shutdown
	SessionTimeout       int64
	Active               bool
	userSessions         *userSessions
	Variants             map[string]*Variant
	sessionPingChannel   chan *variantSessionPing
	sessionDeleteChannel chan []*variantSessionPing
	sessionCookieNames   []string
	config               *config.Config
}

func NewSessions(c *config.Config) *Sessions {
	variants := make(map[string]*Variant)
	for _, variantConfig := range c.Variants {
		variants[variantConfig.Id] = NewVariant(variantConfig)
	}
	us := &Sessions{
		Active: true,
		userSessions: &userSessions{
			m: map[string]map[string]*UserSession{},
		},
		Variants:             variants,
		sessionPingChannel:   make(chan *variantSessionPing),
		sessionDeleteChannel: make(chan []*variantSessionPing),
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
		case sessionPingsOfDeath := <-us.sessionDeleteChannel:
			us.userSessions.Lock()
			for _, sessionPingOfDeath := range sessionPingsOfDeath {
				//fmt.Println("deleting", sessionPingOfDeath.SessionId, len(us.userSessions.m[sessionPingOfDeath.CookieName]))
				delete(us.userSessions.m[sessionPingOfDeath.CookieName], sessionPingOfDeath.SessionId)
			}
			us.userSessions.Unlock()
		case sessionPing := <-us.sessionPingChannel:
			if len(sessionPing.SessionId) > 0 {
				us.userSessions.Lock()
				_, ok := us.userSessions.m[sessionPing.CookieName]
				if !ok {
					us.userSessions.m[sessionPing.CookieName] = map[string]*UserSession{}
				}
				session, ok := us.userSessions.m[sessionPing.CookieName][sessionPing.SessionId]
				if !ok {
					session = NewUserSession(sessionPing)
					us.userSessions.m[sessionPing.CookieName][sessionPing.SessionId] = session
				}
				us.userSessions.Unlock()
				session.LastVisit = time.Now().Unix()
				session.Pageviews++
			}
		}
	}
}

func (s *Sessions) extractSessionId(incomingRequest *http.Request) (sessionID string, cookieName string) {
	for _, cookieName = range s.sessionCookieNames {
		cookie, err := incomingRequest.Cookie(cookieName)
		if err == nil && cookie != nil && len(cookie.Value) > 0 {
			return cookie.Value, cookieName
		}
	}
	return "", ""
}

// GetExistingUserVariant get an existing user variant
func (us *Sessions) GetExistingUserVariant(incomingRequest *http.Request) *Variant {
	sessionID, cookieName := us.extractSessionId(incomingRequest)
	if len(sessionID) > 0 {
		return us.getVariantForUserSessionId(sessionID, cookieName)
	}
	return nil
}

func (us *Sessions) GetBalancedRandomVariant() *Variant {

	variantID := getBalancedRandomVariantId(getVariantStats(us.Variants, us.userSessions, us.SessionTimeout))
	variant, _ := us.Variants[variantID]
	return variant
}

func (us *Sessions) serveVariant(variant *Variant, w http.ResponseWriter, incomingRequest *http.Request) (sessionID string, err error) {
	sessionID, cookieName, err := variant.Serve(w, incomingRequest)
	if err == nil && len(sessionID) > 0 {
		us.sessionPingChannel <- &variantSessionPing{
			SessionId:  sessionID,
			VariantId:  variant.Id,
			CookieName: cookieName,
		}
	}
	return
}
func (us *Sessions) ServeVariant(variant *Variant, w http.ResponseWriter, incomingRequest *http.Request) (err error) {
	_, err = us.serveVariant(variant, w, incomingRequest)
	return err
}
