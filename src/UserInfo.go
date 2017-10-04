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

func (u *UserInfo) CheckPass(password string) bool {
    return u.password == password
}

func (user *UserInfo) Follow(newFollow *UserInfo) bool {
    if newFollow == nil || user.following[newFollow.Username] != nil {
        return false
    }
    user.following[newFollow.Username] = newFollow
    return true
}

func (user *UserInfo) UnFollow(oldFollow *UserInfo) bool {
    if oldFollow == nil || user.following[oldFollow.Username] == nil {
        return false
    }
    user.following[oldFollow.Username] = nil
    return true
}
