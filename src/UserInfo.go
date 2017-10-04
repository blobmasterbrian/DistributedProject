package src

type UserInfo struct {
	Username string
	password string
	following map[string]*UserInfo
}

func NewUserInfo(username, password string) *UserInfo {
    newUser := new(UserInfo)
    newUser.Username = username
    newUser.password = password
    newUser.following = make(map[string]*UserInfo)
    return newUser
}

func (u *UserInfo) CheckPass(password string) bool{
    return u.password == password
}
