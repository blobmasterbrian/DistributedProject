package main

import(
    "fmt"
    "net/http"
    src "../DistributedProject/src"
)

var USERS map[string]src.UserInfo

func main(){

    http.HandleFunc("/signupsubmit", signup)
    http.ListenAndServe(":8080", nil)
}

func welcome(w http.ResponseWriter, r *http.Request) {
}

func signup(w http.ResponseWriter, r *http.Request) {
    if r.Method == http.MethodPost {
        r.ParseForm()
        fmt.Fprintf(w, "username check: %s, password check: %s\n", r.PostFormValue("username"),
            r.PostFormValue("password"))
    }
}

func login(w http.ResponseWriter, r *http.Request) {
    if r.Method == http.MethodPost {
        r.ParseForm()
    }
}
