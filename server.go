package main

import(
    "html/template"
    "fmt"
    "net/http"
    . "../DistributedProject/src"
    "time"
)

var USERS = map[string]*UserInfo{}
const LOGINCOOKIE = "loginCookie"

func main(){
    http.HandleFunc("/", welcomeRedirect)
    http.HandleFunc("/welcome", welcome)
    http.HandleFunc("/signup", signup)
    http.HandleFunc("/login", login)
    http.HandleFunc("/home",home)
    http.HandleFunc("/error", errorPage)
    http.HandleFunc("/follow",follow)

    http.HandleFunc("/signup-response", signupResponse)
    http.HandleFunc("/login-response", loginResponse)
    http.HandleFunc("/search-response",searchResponse)
    http.ListenAndServe(":8080", nil)
}


func welcomeRedirect(w http.ResponseWriter, r *http.Request) {
    http.Redirect(w, r, "/welcome", http.StatusSeeOther)
}

func welcome(w http.ResponseWriter, r *http.Request) {
    exists, _ := loggedInRedirect(w, r)
    if exists {
        http.Redirect(w, r, "/home", http.StatusSeeOther)
        return
    }
    http.ServeFile(w, r, "web/welcome.html")
}

func signup(w http.ResponseWriter, r *http.Request) {
    exists, _ := loggedInRedirect(w, r)
    if exists {
        http.Redirect(w, r, "/home", http.StatusSeeOther)
        return
    }
    http.ServeFile(w, r, "web/signup.html")
}

func login(w http.ResponseWriter, r *http.Request) {
    exists, _ := loggedInRedirect(w, r)
    if exists {
        http.Redirect(w, r, "/home", http.StatusSeeOther)
        return
    }
    http.ServeFile(w, r, "web/login.html")
}

func home(w http.ResponseWriter, r *http.Request) {
    exists, cookie := loggedInRedirect(w, r)
    if !exists {
        http.Redirect(w, r, "/welcome", http.StatusSeeOther)
        return
    }
    t, _ := template.ParseFiles("web/homepage.html")
    t.Execute(w, cookie.Value)
}

func errorPage(w http.ResponseWriter, r *http.Request) {
    t, _ := template.ParseFiles("web/error.html")
    t.Execute(w, struct {Name string; Error string}{Name: "Dave", Error: "Singularity"})
}

func follow(w http.ResponseWriter, r *http.Request) {
    exists, _ := loggedInRedirect(w, r)
    if !exists {
        http.Redirect(w, r, "/welcome", http.StatusSeeOther)
        return
    }
    if r.Method == http.MethodGet{
        r.ParseForm()
        fmt.Println("I made it")
        fmt.Println(r.FormValue("username"))
    }
}

func signupResponse(w http.ResponseWriter, r *http.Request) {
    if r.Method == http.MethodPost {
        r.ParseForm()
        if (r.PostFormValue("password") != r.PostFormValue("confirm")) || USERS[r.PostFormValue("username")] != nil {
            http.Redirect(w, r, "/error", http.StatusSeeOther)
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
            http.Redirect(w,r,"/error",http.StatusSeeOther)
        }
    }
}

func searchResponse(w http.ResponseWriter, r *http.Request) {
    exists, _ := loggedInRedirect(w, r)
    if !exists {
        http.Redirect(w, r, "/welcome", http.StatusSeeOther)
        return
    }
    if r.Method == http.MethodGet{
        r.ParseForm()
        if USERS[r.FormValue("username")] != nil {
            t, _ := template.ParseFiles("web/searchResult.html")
            t.Execute(w, struct{Username string; Link string}{Username: r.FormValue("username"), Link: "temp"})
        }
    }
}

func loggedInRedirect(w http.ResponseWriter, r *http.Request) (LoggedIn bool, Cookie *http.Cookie) {
    cookie, err := r.Cookie(LOGINCOOKIE)
    if err != nil {
        fmt.Println(err)
    }
    if cookie == nil {
        return false, nil
    }
    return true, cookie
}

func genCookie(username string) *http.Cookie {
    return &http.Cookie{
        Name:     LOGINCOOKIE,
        Value:    username,
        Expires:  time.Now().Add(24 * time.Hour),
    }
}