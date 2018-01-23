package variantbalancer

import (
	"errors"
	"net/http"

	"github.com/foomo/variant-balancer/config"
	us "github.com/foomo/variant-balancer/usersessions"
	"github.com/foomo/variant-balancer/context"
)

type RandomVariantResolver interface {
	ResolveRandomVariantFor(userSessions *us.Sessions, req *http.Request) *us.Variant
}

type Balancer struct {
	//
	UserSessions          []*us.Sessions
	Service               *Service
	RandomVariantResolver RandomVariantResolver
}

func NewBalancer() *Balancer {
	b := &Balancer{
		UserSessions: []*us.Sessions{},
		Service:      new(Service),
	}
	b.Service.balancer = b
	return b
}

func (b *Balancer) RunSession(c *config.Config, flushSessions bool) {
	userSessions := us.NewSessions(c)
	if flushSessions {
		userSessions.Active = true
		oldSessions := b.UserSessions
		b.UserSessions = []*us.Sessions{userSessions}
		for _, oldSession := range oldSessions {
			oldSession.Active = false
		}
	} else {
		b.UserSessions = append(b.UserSessions, userSessions)
		for i, userSessions := range b.UserSessions {
			userSessions.Active = i == len(b.UserSessions)-1
		}

	}
}

func (b *Balancer) GetUserSessionsStatus() []*us.SessionsStatus {
	sessionsStatus := []*us.SessionsStatus{}
	for _, us := range b.UserSessions {
		sessionsStatus = append(sessionsStatus, us.GetStatus())
	}
	return sessionsStatus
}

func (b *Balancer) ServeHTTP(w http.ResponseWriter, incomingRequest *http.Request) error {
	context.Initialize(incomingRequest)
	if len(b.UserSessions) > 0 {
		for _, sessions := range b.UserSessions {
			variant := sessions.GetExistingUserVariant(incomingRequest)
			if variant == nil && sessions.Active {
				if b.RandomVariantResolver != nil {
					variant = b.RandomVariantResolver.ResolveRandomVariantFor(sessions, incomingRequest)
				}
				if variant == nil {
					variant = sessions.GetBalancedRandomVariant()
				}
			}
			if variant != nil {
				return sessions.ServeVariant(variant, w, incomingRequest)
			}
		}
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("503 - no variant to serve"))
		return errors.New("no variant to serve")
	}
	w.WriteHeader(http.StatusServiceUnavailable)
	w.Write([]byte("503 - not available at the moment (no config?)"))
	return errors.New("not available at the moment (no config?)")
}
