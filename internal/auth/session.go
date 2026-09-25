package auth

import "time"

const SessionDuration = 7 * 24 * time.Hour

func SessionExpiresAt() time.Time {
	return time.Now().Add(SessionDuration)
}