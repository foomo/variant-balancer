package usersessions

import (
	"time"
)

type UserSession struct {
	Pageviews int64
	LastVisit int64
	Id        string
	VariantId string
}

func NewUserSession(sessionPing *VariantSessionPing) *UserSession {
	return &UserSession{
		Pageviews: 0,
		LastVisit: 0,
		Id:        sessionPing.SessionId,
		VariantId: sessionPing.VariantId,
	}
}

func (s *UserSession) IsActive(timeout int64) bool {
	return s.Pageviews > 1 && time.Now().Unix()-s.LastVisit > timeout
}
