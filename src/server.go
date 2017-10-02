package main

import(
    "fmt"
    "net/http"
)

func main(){
    http.HandleFunc("/signupsubmit", signup)
    http.ListenAndServe(":8080", nil)
}

func signup(w http.ResponseWriter, r *http.Request) {
    if r.Method == http.MethodPost {
        r.ParseForm()
        fmt.Fprintf(w, "username check: %s, password check: %s\n", r.PostFormValue("username"),
            r.PostFormValue("password"))
    }
}


