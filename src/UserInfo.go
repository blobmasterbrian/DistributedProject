package src

import (
    "time"
    "container/heap"
)


type UserInfo struct {
	Username   string
	password   string
	following  map[string]*UserInfo
    posts      []Post
}

func NewUserInfo(username, password string) *UserInfo {
    newUser := new(UserInfo)
    newUser.Username = username
    newUser.password = password
    newUser.following = make(map[string]*UserInfo)
    return newUser
}

type Post struct {
    Message  string
    Poster   string
    Time     string
    stamp    time.Time
    index    int  //index of post in priority queue
}

type PriorityQueue []*Post

func (q PriorityQueue) Len() int {return len(q)}

func (q PriorityQueue) Less(i, j int) bool {
    return q[j].stamp.Before(q[i].stamp)
}

func (q PriorityQueue) Swap(i,j int) {
    q[i], q[j] = q[j], q[i]
    q[i].index = i
    q[j].index = j
}

func (q *PriorityQueue) Push(x interface{}){
    newLen := len(*q)
    newPost := x.(*Post)
    newPost.index = newLen
    *q = append(*q, newPost)
}

func (q *PriorityQueue) Pop() interface{} {
    oldQ := *q
    n := len(oldQ)
    removedPost := oldQ[n-1]
    removedPost.index = -1
    *q = oldQ[0 : n-1]
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
    newPost := Post{Message: msg, Poster: user.Username, Time: time.RFC1123[0:len(time.RFC1123)-4], stamp: time.Now()}
    user.posts = append(user.posts, newPost)
}

func (user *UserInfo) GetAllChirps() []Post {
    var result = []Post{}

    var allChirps PriorityQueue
    heap.Init(&allChirps)
    for _, followed := range user.following {
        for i, _ := range followed.posts {
            heap.Push(&allChirps, &(followed.posts[i]))
        }
    }
    for allChirps.Len() > 0 {
        x := *heap.Pop(&allChirps).(*Post)
        result = append(result, x)
    }
    return result
}
