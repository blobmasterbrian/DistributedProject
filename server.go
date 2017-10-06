package main

import(
    "html/template"
    "fmt"
    "net/http"
    . "../DistributedProject/src"
    "time"
)

var USERS = map[string]*UserInfo{}  // Map of all users
const LOGIN_COOKIE = "loginCookie"  // Cookie to keep users logged in

func main(){
    http.HandleFunc("/", welcomeRedirect)   // function for server address page
    http.HandleFunc("/welcome", welcome)    // function for welcome page (main page for not logged in users)
    http.HandleFunc("/signup", signup)      // function for signup page
    http.HandleFunc("/login", login)        // function for login page
    http.HandleFunc("/logout", logout)      // function for logout page
    http.HandleFunc("/home", home)          // function for home page (main page for logged in users)
    http.HandleFunc("/error", errorPage)    // function for error page

    http.HandleFunc("/follow", follow)                   // function for follow submission
    http.HandleFunc("/unfollow", unfollow)               // function for unfollow submission
    http.HandleFunc("/submit-post", submitPost)          // function for post submission
    http.HandleFunc("/signup-response", signupResponse)  // function for signup submission
    http.HandleFunc("/login-response", loginResponse)    // function for login submission
    http.HandleFunc("/search-response", searchResponse)  // function for search submission
    http.HandleFunc("/delete-account", deleteAccount)    // function for account deletion submission
    http.ListenAndServe(":8080", nil)
}


func welcomeRedirect(w http.ResponseWriter, r *http.Request) {
    http.Redirect(w, r, "/welcome", http.StatusSeeOther)  // URL always displays welcome
}

func welcome(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
    w.Header().Set("Pragma", "no-cache")
    w.Header().Set("Expires", "0")
    exists, _ := getCookie(w, r)
    if exists {
        http.Redirect(w, r, "/home", http.StatusSeeOther)  // redirect to home if the user is already logged in
        return
    }
    http.ServeFile(w, r, "web/welcome.html")
}

func signup(w http.ResponseWriter, r *http.Request) {
    exists, _ := getCookie(w, r)
    if exists {
        http.Redirect(w, r, "/home", http.StatusSeeOther)
        return
    }
    http.ServeFile(w, r, "web/signup.html")
}

func login(w http.ResponseWriter, r *http.Request) {
    exists, _ := getCookie(w, r)
    if exists {
        http.Redirect(w, r, "/home", http.StatusSeeOther)
        return
    }
    http.ServeFile(w, r, "web/login.html")
}

func logout(w http.ResponseWriter, r *http.Request) {
    cookie, _ := r.Cookie(LOGIN_COOKIE)
    cookie.MaxAge = -1
    cookie.Expires = time.Now().Add(-1 * time.Hour)
    http.SetCookie(w, cookie)
    http.Redirect(w, r, "/welcome", http.StatusSeeOther)
}

func home(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
    w.Header().Set("Pragma", "no-cache")
    w.Header().Set("Expires", "0")
    exists, cookie := getCookie(w, r)
    if !exists {
        http.Redirect(w, r, "/welcome", http.StatusSeeOther)
        return
    }
    t, err := template.ParseFiles("web/homepage.html")
    if err != nil {
        fmt.Println(err)
    }
    t.Execute(w, struct{
            Username string
            Posts []Post
        }{
            cookie.Value,
            USERS[cookie.Value].GetAllChirps(),
        })
}

func errorPage(w http.ResponseWriter, r *http.Request) {
    t, err := template.ParseFiles("web/error.html")
    if err != nil {
        fmt.Println(err)
    }
    t.Execute(w, struct{Username string; Error string}{Username: "Dave", Error: "Singularity"})
}

//the current user (determined by the cookie) will add a new user to their followed list
//based on form value, if follow fails redirect to the error pagae
func follow(w http.ResponseWriter, r *http.Request) {
    exists, cookie := getCookie(w, r)
    if !exists {
        http.Redirect(w, r, "/welcome", http.StatusSeeOther)
        return
    }
    if r.Method == http.MethodPost{
        r.ParseForm()
        if USERS[cookie.Value] == nil {
            return
        }
        if !USERS[cookie.Value].Follow(USERS[r.PostFormValue("username")]){
            http.Redirect(w,r, "/error", http.StatusSeeOther)
        } else {
            http.Redirect(w,r,"/home", http.StatusSeeOther)
        }
    }
}

//reverse logic of follow
func unfollow(w http.ResponseWriter, r *http.Request){
    exists, cookie := getCookie(w, r)
    if !exists {
        http.Redirect(w, r, "/welcome", http.StatusSeeOther)
        return
    }
    if r.Method == http.MethodPost {
        r.ParseForm()
        if USERS[cookie.Value] == nil {
            return
        }
        if !USERS[cookie.Value].UnFollow(USERS[r.PostFormValue("username")]){
            http.Redirect(w, r, "/error", http.StatusSeeOther)
        } else {
            http.Redirect(w, r, "/home", http.StatusSeeOther)
        }
    }

}

//reads a post from form input, then appends it to the slice of posts per user
func submitPost(w http.ResponseWriter, r *http.Request) {
    exists, cookie := getCookie(w,r)
    if !exists || USERS[cookie.Value] == nil{
        http.Redirect(w, r, "/welcome", http.StatusSeeOther)
        return
    }
    if r.Method == http.MethodPost {
        r.ParseForm()
        USERS[cookie.Value].WritePost(r.PostFormValue("post"))
        http.Redirect(w, r, "/home", http.StatusSeeOther)
    }
}

//creates a new user if the provided username is not already taken
func signupResponse(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
    w.Header().Set("Pragma", "no-cache")
    w.Header().Set("Expires", "0")

    if r.Method == http.MethodPost {
        r.ParseForm()
        if (r.PostFormValue("password") != r.PostFormValue("confirm")) || USERS[r.PostFormValue("username")] != nil {
            http.Redirect(w, r, "/signup", http.StatusSeeOther)
            return
        }
        USERS[r.PostFormValue("username")] = NewUserInfo(r.PostFormValue("username"), r.PostFormValue("password"))

        http.SetCookie(w, genCookie(r.PostFormValue("username")))
        http.Redirect(w, r, "/home", http.StatusSeeOther)
        fmt.Printf("Username: %s, Password: %s, Confirmed Pass: %s\n",
            USERS[r.PostFormValue("username")].Username,
            r.PostFormValue("password"),
            r.PostFormValue("confirm"))
    }
}

func loginResponse(w http.ResponseWriter, r *http.Request) {
    if r.Method == http.MethodPost {
        r.ParseForm()

        if USERS[r.PostFormValue("username")] != nil &&
            USERS[r.PostFormValue("username")].CheckPass(r.PostFormValue("password")) {

            http.SetCookie(w, genCookie(r.PostFormValue("username")))
            http.Redirect(w, r, "/home", http.StatusSeeOther)
            fmt.Printf("Username: %s, Password: %s\n", r.PostFormValue("username"),
                r.PostFormValue("password"))
        } else {
            fmt.Println(USERS[r.PostFormValue("username")])
            http.Redirect(w,r,"/login",http.StatusSeeOther)
        }
    }
}

//searches for a user, provides user info if the user did not search for him/herself
//provides a link to follow/unfollow based on current follow status
func searchResponse(w http.ResponseWriter, r *http.Request) {
    exists, cookie := getCookie(w, r)
    if !exists {
        http.Redirect(w, r, "/welcome", http.StatusSeeOther)
        return
    }
    if r.Method == http.MethodGet{
        r.ParseForm()
        if USERS[r.FormValue("username")] != nil && r.FormValue("username") != cookie.Value {
            t, _ := template.ParseFiles("web/searchResult.html")
            isFollowing := USERS[cookie.Value].IsFollowing(USERS[r.FormValue("username")])
            var followStr string
            if isFollowing {
                followStr = "unfollow"
            } else {
                followStr = "follow"
            }
            t.Execute(w, struct{Username, Follow string}{Username: r.FormValue("username"), Follow: followStr})
        } else {
            http.Redirect(w, r, "/home", http.StatusSeeOther)
        }
    }
}

func deleteAccount(w http.ResponseWriter, r *http.Request) {
    cookie, _ := r.Cookie(LOGIN_COOKIE)
    user := USERS[cookie.Value]
    for i := range USERS {
        if USERS[i] != nil && user != nil{
            USERS[i].UnFollow(user)
        }
    }
    USERS[cookie.Value] = nil
    cookie.MaxAge = -1
    cookie.Expires = time.Now().Add(-1 * time.Hour)
    http.SetCookie(w, cookie)
    http.Redirect(w, r, "/welcome", http.StatusSeeOther)
}

func getCookie(w http.ResponseWriter, r *http.Request) (LoggedIn bool, Cookie *http.Cookie) {
    //ignoring error value because it is likely that the cookie might not exist here
    cookie, _ := r.Cookie(LOGIN_COOKIE)
    if cookie == nil {
        return false, nil
    }
    return true, cookie
}

func genCookie(username string) *http.Cookie {
    return &http.Cookie{
        Name:     LOGIN_COOKIE,
        Value:    username,
        Expires:  time.Now().Add(24 * time.Hour),
    }
}
