package stylometry_session

import "sync"

const fixedUserID uint = 1

type CurrentUserInfo struct {
	ID uint
}

var (
	currentUser     *CurrentUserInfo
	currentUserOnce sync.Once
)

func CurrentUser() *CurrentUserInfo {
	currentUserOnce.Do(func() {
		currentUser = &CurrentUserInfo{
			ID: fixedUserID,
		}
	})

	return currentUser
}
