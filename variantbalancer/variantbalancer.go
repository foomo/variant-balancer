package variantbalancer

import (
	"errors"
	"github.com/foomo/variant-balancer/config"
	us "github.com/foomo/variant-balancer/usersessions"
	"net/http"
)

type Balancer struct {
	//
	UserSessions []*us.Sessions
	Service      *Service
}

func NewBalancer() *Balancer {
	b := &Balancer{
		UserSessions: []*us.Sessions{},
		Service:      new(Service),
	}
	b.Service.balancer = b
	return b
}

func (b *Balancer) RunSession(c *config.Config) {
	userSessions := us.NewSessions(c)
	b.UserSessions = append(b.UserSessions, userSessions)
	for i, userSessions := range b.UserSessions {
		userSessions.Active = i == len(b.UserSessions)-1
	}
}

func (b *Balancer) GetUserSessionsStatus() []*us.SessionsStatus {
	sessionsStatus := []*us.SessionsStatus{}
	for _, us := range b.UserSessions {
		sessionsStatus = append(sessionsStatus, us.GetStatus())
	}
	return sessionsStatus
}

func (b *Balancer) ServeHTTP(w http.ResponseWriter, incomingRequest *http.Request, cacheId string) error {
	if len(b.UserSessions) > 0 {
		for _, sessions := range b.UserSessions {
			variant := sessions.GetExistingUserVariant(incomingRequest)
			if variant == nil && sessions.Active {
				variant = sessions.GetBalancedRandomVariant()
			}
			if variant != nil {
				return sessions.ServeVariant(variant, w, incomingRequest, cacheId)
			}
		}
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("503 - no variant to serve"))
		return errors.New("no variant to serve")
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("503 - not available at the moment (no config?)"))
		return errors.New("not available at the moment (no config?)")
	}
}
