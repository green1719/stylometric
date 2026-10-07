package stylometry_session

import "sync"

const stylometrySampleFixedUserID uint = 1

type StylometrySampleCurrentUser struct {
	StylometrySampleID uint
}

var (
	stylometrySampleCurrentUser     *StylometrySampleCurrentUser
	stylometrySampleCurrentUserOnce sync.Once
)

func CurrentUser() *StylometrySampleCurrentUser {
	stylometrySampleCurrentUserOnce.Do(func() {
		stylometrySampleCurrentUser = &StylometrySampleCurrentUser{
			StylometrySampleID: stylometrySampleFixedUserID,
		}
	})

	return stylometrySampleCurrentUser
}
