package usersessions

type UserSession struct {
	Pageviews int64
	LastVisit int64
	Id        string
	VariantId string
}

func NewUserSession(sessionPing *variantSessionPing) *UserSession {
	return &UserSession{
		Pageviews: 0,
		LastVisit: 0,
		Id:        sessionPing.SessionId,
		VariantId: sessionPing.VariantId,
	}
}
