package lib

import (
    "time"
    "container/heap"
    "sync"
)

// Struct to hold all necessary user info
type UserInfo struct {
    Username   string
    Password   string
    Following  map[string]bool
    FollowedBy []string
    Posts      []Post
    mut        *sync.Mutex
}

// NOTE: no longer necessary as it is no longer package private, creates new UserInfo struct
func NewUserInfo(username, password string) *UserInfo {
    newUser := new(UserInfo)
    newUser.Username = username
    newUser.Password = password
    newUser.Following = make(map[string]bool)
    newUser.mut = &sync.Mutex{}
    return newUser
}

// Struct to hold data associated with a user's post
type Post struct {
    Poster  string
    Message string
    Time    string
    Stamp   time.Time
    Index   int  //index of post in priority queue
}

type PriorityQueue []*Post  // typedef of PriorityQueue as slice of Post pointers

// Functions below are required for implementing a heap interface, used for getting all follower's posts in order

// Gets length of PriorityQueue
func (q PriorityQueue) Len() int {return len(q)}

// Implements comparison function
func (q PriorityQueue) Less(i, j int) bool {
    return q[j].Stamp.Before(q[i].Stamp)
}

// Swaps elements when called by PriorityQueue
func (q PriorityQueue) Swap(i,j int) {
    q[i], q[j] = q[j], q[i]
    q[i].Index = i
    q[j].Index = j
}

// Implementation of pushing into the queue
func (q *PriorityQueue) Push(x interface{}){
    newLen := len(*q)
    newPost := x.(*Post)
    newPost.Index = newLen
    *q = append(*q, newPost)
}

// Implementation of poping from the queue
func (q *PriorityQueue) Pop() interface{} {
    oldQ := *q
    n := len(oldQ)
    removedPost := oldQ[n-1]
    removedPost.Index = -1
    *q = oldQ[0 : n-1]
    return removedPost
}

// Locks the user mutex
func (user *UserInfo) Lock() {
    user.mut.Lock()
}

// Unlocks the user mutex
func (user *UserInfo) Unlock() {
    user.mut.Unlock()
}

// Checks if a given password hash matches the password hash stored in UserInfo
func (user *UserInfo) CheckPass(password string) bool {
    user.mut.Lock()
    defer user.mut.Unlock()
    return user.Password == password
}

// Current UserInfo follows the UserInfo passed in parameter
func (user *UserInfo) Follow(newFollow *UserInfo) bool {
    user.mut.Lock()
    newFollow.mut.Lock()
    defer user.mut.Unlock()
    defer newFollow.mut.Unlock()
    if newFollow == nil || user.Following[newFollow.Username] {
        return false
    }
    newFollow.FollowedBy = append(newFollow.FollowedBy, user.Username)
    user.Following[newFollow.Username] = true
    return true
}

// Current UserInfo unfollows the UserInfo passed in parameter
func (user *UserInfo) UnFollow(oldFollow *UserInfo) bool {
    user.mut.Lock()
    oldFollow.mut.Lock()
    defer user.mut.Unlock()
    defer oldFollow.mut.Unlock()
    if oldFollow == nil || !user.Following[oldFollow.Username] {
        return false
    }
    for i := range oldFollow.FollowedBy {
        if oldFollow.FollowedBy[i] == user.Username {
            oldFollow.FollowedBy = append(oldFollow.FollowedBy[:i], oldFollow.FollowedBy[i+1:]...)
            break
        }
    }
    delete(user.Following, oldFollow.Username)
    return true
}

// Checks if current UserInfo is following the UserInfo passed in parameter
func (user *UserInfo) IsFollowing(other *UserInfo) bool {
    user.mut.Lock()
    other.mut.Lock()
    defer user.mut.Unlock()
    defer other.mut.Unlock()
    for item := range user.Following {
        if item == other.Username {
            return true
        }
    }
    return false
}


// Creates a Post appended to UserInfo's Posts member
func (user *UserInfo) WritePost(msg string){
    user.mut.Lock()
    newPost := Post{Poster: user.Username, Message: msg, Time: time.Now().Format(time.RFC1123)[0:len(time.RFC1123)-4], Stamp: time.Now()}
    user.Posts = append(user.Posts, newPost)
    user.mut.Unlock()
}

// Creates a PriorityQueue implemented with a heap to pull all of the posts and return a slice with
// the posts in order (includes the current user's posts)
func (user *UserInfo) GetAllChirps(USERS map[string]*UserInfo) []Post {
    user.mut.Lock()
    defer user.mut.Unlock()
    var result = []Post{}

    var allChirps PriorityQueue
    heap.Init(&allChirps)  // initializes the PriorityQueue as a heap
    for i := range user.Posts {
        heap.Push(&allChirps, &(user.Posts[i]))  // uses the Push method defined above
    }
    for followed := range user.Following {
        USERS[followed].mut.Lock()
        for i := range USERS[followed].Posts {
            heap.Push(&allChirps, &(USERS[followed].Posts[i]))  // uses the Push method defined above
        }
        USERS[followed].mut.Unlock()
    }
    for allChirps.Len() > 0 {  // uses the Len method defined above
        result = append(result, *heap.Pop(&allChirps).(*Post))  // appends the result of Pop (defined above) to the resulting slice
    }
    return result
}
