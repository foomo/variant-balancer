package usersessions

func getBalancedRandomVariantId(variantStats map[string]*SessionStats) (variantId string) {
	distances := make(map[string]float64)
	for id, stats := range variantStats {
		if stats.Share > 0.0 {
			distances[id] = stats.Share - stats.ActiveShare
		}
	}
	d := float64(-100000000000)
	for id, distance := range distances {
		if distance > d {
			d = distance
			variantId = id
		}
	}
	return variantId
}

func (us *Sessions) getVariantForUserSessionId(sessionId string, cookieName string) *Variant {
	session, ok := us.UserSessions[cookieName][sessionId]
	if ok {
		variant, ok := us.Variants[session.VariantId]
		if ok {
			return variant
		}
	}
	return nil
}
