package lib

import (
    "time"
    "container/heap"
)

type UserInfo struct {
    Username   string
    Password   string
    Following  map[string]*UserInfo
    FollowedBy []*UserInfo
    Posts      []Post
}

// NOTE: no longer necessary as it is no longer package private
func NewUserInfo(username, password string) *UserInfo {
    newUser := new(UserInfo)
    newUser.Username = username
    newUser.Password = password
    newUser.Following = make(map[string]*UserInfo)
    return newUser
}

type Post struct {
    Poster  string
    Message string
    Time    string
    Stamp   time.Time
    Index   int  //index of post in priority queue
}

type PriorityQueue []*Post

//below functions required for implementing a heap interface, used in getting all follower's
//posts in order
func (q PriorityQueue) Len() int {return len(q)}

func (q PriorityQueue) Less(i, j int) bool {
    return q[j].Stamp.Before(q[i].Stamp)
}

func (q PriorityQueue) Swap(i,j int) {
    q[i], q[j] = q[j], q[i]
    q[i].Index = i
    q[j].Index = j
}

func (q *PriorityQueue) Push(x interface{}){
    newLen := len(*q)
    newPost := x.(*Post)
    newPost.Index = newLen
    *q = append(*q, newPost)
}

func (q *PriorityQueue) Pop() interface{} {
    oldQ := *q
    n := len(oldQ)
    removedPost := oldQ[n-1]
    removedPost.Index = -1
    *q = oldQ[0 : n-1]
    return removedPost
}

func (u *UserInfo) CheckPass(password string) bool {
    return u.Password == password
}

func (user *UserInfo) Follow(newFollow *UserInfo) bool {
    if newFollow == nil || user.Following[newFollow.Username] != nil {
        return false
    }
    newFollow.FollowedBy = append(newFollow.FollowedBy, user)
    user.Following[newFollow.Username] = newFollow
    return true
}

func (user *UserInfo) UnFollow(oldFollow *UserInfo) bool {
    if oldFollow == nil || user.Following[oldFollow.Username] == nil {
        return false
    }
    for i := range oldFollow.FollowedBy {
        if oldFollow.FollowedBy[i] == user {
            oldFollow.FollowedBy = append(oldFollow.FollowedBy[:i], oldFollow.FollowedBy[i+1:]...)
            break
        }
    }
    delete(user.Following, oldFollow.Username)
    return true
}

func (user *UserInfo) IsFollowing(other *UserInfo) bool {
    for i := range user.Following {
        if user.Following[i] == other {
            return true
        }
    }
    return false
}

func (user *UserInfo) deleteAccount() {
    for _, other := range user.FollowedBy{
        other.UnFollow(user)
    }
}

func (user *UserInfo) WritePost(msg string){
    newPost := Post{Poster: user.Username, Message: msg, Time: time.Now().Format(time.RFC1123)[0:len(time.RFC1123)-4], Stamp: time.Now()}
    user.Posts = append(user.Posts, newPost)
}

//creates a priority queue implemented with a heap to pull all of the posts and return a slice with
//the posts in order. includes the current user's posts
func (user *UserInfo) GetAllChirps() []Post {
    var result = []Post{}

    var allChirps PriorityQueue
    heap.Init(&allChirps)
    for i := range user.Posts {
        heap.Push(&allChirps, &(user.Posts[i]))
    }
    for _, followed := range user.Following {
        if followed != nil {
            for i := range followed.Posts {
                heap.Push(&allChirps, &(followed.Posts[i]))
            }
        }
    }
    for allChirps.Len() > 0 {
        result = append(result, *heap.Pop(&allChirps).(*Post))
    }
    return result
}
