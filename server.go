package main

import(
    "html/template"
    "fmt"
    "net/http"
    . "../DistributedProject/src"
)

var USERS map[string]*UserInfo

func main(){
    serverInit()

    http.HandleFunc("/", welcomeRedirect)
    http.HandleFunc("/welcome", welcome)
    http.HandleFunc("/signup", signup)
    http.HandleFunc("/login", login)
    http.HandleFunc("/home",home)
    http.HandleFunc("/error", errorPage)

    http.HandleFunc("/signup-response", signupResponse)
    http.HandleFunc("/login-response", loginResponse)
    http.HandleFunc("/search-response",searchResponse)
    http.ListenAndServe(":8080", nil)
}

func serverInit(){
    USERS = make(map[string]*UserInfo)
}

func welcomeRedirect(w http.ResponseWriter, r *http.Request) {
    http.Redirect(w, r, "/welcome", 308 )
}

func welcome(w http.ResponseWriter, r *http.Request) {
    http.ServeFile(w, r, "web/welcome.html")
}

func signup(w http.ResponseWriter, r *http.Request) {
    http.ServeFile(w, r, "web/signup.html")
}

func login(w http.ResponseWriter, r *http.Request) {
    http.ServeFile(w, r, "web/login.html")
}

func home(w http.ResponseWriter, r *http.Request) {
    t, _ := template.ParseFiles("web/homepage.html")
    t.Execute(w, "Charlie")
}

func errorPage(w http.ResponseWriter, r *http.Request) {
    t, _ := template.ParseFiles("web/error.html")
    t.Execute(w, struct {Name string; Error string}{Name: "Dave", Error: "Singularity"})
}

func signupResponse(w http.ResponseWriter, r *http.Request) {
    if r.Method == http.MethodPost {
        r.ParseForm()
        if (r.PostFormValue("password") != r.PostFormValue("confirm")) || USERS[r.PostFormValue("username")] != nil {
            http.Redirect(w, r, "/error", 308)
            return
        }
        newUser := &UserInfo{Username:r.PostFormValue("username"), Password:r.PostFormValue("password")}
        USERS[r.PostFormValue("username")] = newUser
        http.Redirect(w, r, "/home", 308)
        fmt.Printf("Username: %s, Password: %s, Confirmed Pass: %s\n",
            USERS[r.PostFormValue("username")].Username,
            USERS[r.PostFormValue("username")].Password,
            r.PostFormValue("confirm"))
    }
}

func loginResponse(w http.ResponseWriter, r *http.Request) {
    if r.Method == http.MethodPost {
        r.ParseForm()

        if USERS[r.PostFormValue("username")] != nil &&
            USERS[r.PostFormValue("username")].Password == r.PostFormValue("password") {

            http.Redirect(w, r, "/home", 308)
            fmt.Printf("Username: %s, Password: %s\n", r.PostFormValue("username"),
                r.PostFormValue("password"))
        } else{
            fmt.Println(USERS[r.PostFormValue("username")])
            http.Redirect(w,r,"/error",308)
        }
    }
}

func searchResponse(w http.ResponseWriter, r *http.Request){
    if r.Method == http.MethodGet{
        r.ParseForm()
        if USERS[r.FormValue("username")] != nil {
            t, _ := template.ParseFiles("web/searchResult.html")
            t.Execute(w, struct{Username string; Link string}{Username: r.FormValue("username"), Link: "temp"})
        }
    }
}
