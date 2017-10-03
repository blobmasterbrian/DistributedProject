package main

import(
    "html/template"
    "fmt"
    "net/http"
    src "../DistributedProject/src"
)

var USERS map[string]src.UserInfo

func main(){
    http.HandleFunc("/", welcomeRedirect)
    http.HandleFunc("/welcome", welcome)
    http.HandleFunc("/signup", signup)
    http.HandleFunc("/login", login)
    http.HandleFunc("/home",home)
    http.HandleFunc("/error", errorPage)

    http.HandleFunc("/signup-response", signupResponse)
    http.HandleFunc("/login-response", loginResponse)
    http.ListenAndServe(":8080", nil)
}

func welcomeRedirect(w http.ResponseWriter, r *http.Request) {
    http.Redirect(w, r, "/welcome", http.StatusPermanentRedirect)
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
        if r.PostFormValue("password") != r.PostFormValue("confirm") {
            http.Redirect(w, r, "/error", http.StatusPermanentRedirect)
            return
        }
        http.Redirect(w, r, "/home", http.StatusPermanentRedirect)
        fmt.Printf("Username: %s, Password: %s, Confirmed Pass: %s\n", r.PostFormValue("username"),
            r.PostFormValue("password"), r.PostFormValue("confirm"))
    }
}

func loginResponse(w http.ResponseWriter, r *http.Request) {
    if r.Method == http.MethodPost {
        r.ParseForm()
        http.Redirect(w, r, "/home", http.StatusPermanentRedirect)
        fmt.Printf("Username: %s, Password: %s\n", r.PostFormValue("username"),
            r.PostFormValue("password"))
    }
}
