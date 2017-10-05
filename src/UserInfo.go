package src

import (
    "time"
    "container/heap"
)


type UserInfo struct {
	Username string
	password string
	following map[string]*UserInfo
    posts []post
}

func NewUserInfo(username, password string) *UserInfo {
    newUser := new(UserInfo)
    newUser.Username = username
    newUser.password = password
    newUser.following = make(map[string]*UserInfo)
    return newUser
}

type post struct {
    stamp time.Time
    message string
    index int       //index of post in priority queue
}

type PriorityQueue []*post

func (q PriorityQueue) Len() int {return len(q)}

func (q PriorityQueue) Less(i, j int) bool {
    return q[i].stamp.Before(q[j].stamp)
}

func (q PriorityQueue) Swap(i,j int) {
    q[i], q[j] = q[j], q[i]
    q[i].index = i
    q[j].index = j
}

func (q *PriorityQueue) Push(x interface{}){
    newLen := len(*q)
    newPost := x.(*post)
    newPost.index = newLen
    *q = append(*q, newPost)
}

func (q *PriorityQueue) Pop() interface{} {
    oldQ := *q
    n := len(oldQ)
    removedPost := oldQ[n-1]
    removedPost.index = -1
    *q = oldQ[0: n-1]
    return removedPost
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

func (user *UserInfo) WritePost(msg string){
    newPost := post{message: msg, stamp: time.Now()}
    user.posts = append(user.posts, newPost)
}

func (user *UserInfo) GetAllChirps() []post {
    var result []post

    allChirps := make(PriorityQueue, 10)
    heap.Init(&allChirps)
    for _, followed := range user.following {
        for _, singlePost := range followed.posts {
            heap.Push(&allChirps,singlePost)
        }
    }
    for allChirps.Len() > 0 {
        result = append(result, *heap.Pop(&allChirps).(*post))
    }
    return result
}
