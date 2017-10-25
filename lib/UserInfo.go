package src

import (
    "time"
    "container/heap"
)


type UserInfo struct {
	Username   string
	Password   string
	Following  map[string]*UserInfo
    Posts      []Post
}

func NewUserInfo(username, password string) *UserInfo {
    newUser := new(UserInfo)
    newUser.Username = username
    newUser.password = password
    newUser.following = make(map[string]*UserInfo)
    return newUser
}

type Post struct {
    Poster string
    Message  string
    Time     string
    Stamp    time.Time
    Index    int  //index of post in priority queue
}

type PriorityQueue []*Post

//below functions required for implementing a heap interface, used in getting all follower's
//posts in order
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
    delete(user.following, oldFollow.Username)
    return true
}

func (user *UserInfo) IsFollowing(other *UserInfo) bool {
    for i := range user.following {
        if user.following[i] == other {
            return true
        }
    }
    return false
}

func (user *UserInfo) WritePost(msg string){
    newPost := Post{Poster: user.Username, Message: msg, Time: time.Now().Format(time.RFC1123)[0:len(time.RFC1123)-4], stamp: time.Now()}
    user.posts = append(user.posts, newPost)
}

//creates a priority queue implemented with a heap to pull all of the posts and return a slice with
//the posts in order. includes the current user's posts
func (user *UserInfo) GetAllChirps() []Post {
    var result = []Post{}

    var allChirps PriorityQueue
    heap.Init(&allChirps)
    for i := range user.posts {
        heap.Push(&allChirps, &(user.posts[i]))
    }
    for _, followed := range user.following {
        if followed != nil {
            for i := range followed.posts {
                heap.Push(&allChirps, &(followed.posts[i]))
            }
        }
    }
    for allChirps.Len() > 0 {
        result = append(result, *heap.Pop(&allChirps).(*Post))
    }
    return result
}
