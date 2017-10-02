package src

type UserInfo struct {
	Username string
	Password string
	Following []*UserInfo
}
