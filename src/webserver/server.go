package main

import(
    . "../../lib"
    "encoding/binary"
    "encoding/gob"
    "fmt"
    "html/template"
    "net"
    "net/http"
    "time"
)

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
    http.HandleFunc("/search-response", searchResponse)  // function for search submission
    http.HandleFunc("/delete-account", deleteAccount)    // function for account deletion submission
    http.ListenAndServe(":8080", nil)
}


func welcomeRedirect(w http.ResponseWriter, r *http.Request) {
    http.Redirect(w, r, "/welcome", http.StatusSeeOther)  // URL always displays welcome
}

func welcome(w http.ResponseWriter, r *http.Request) {
    clearCache(w)
    exists, _ := getCookie(r)
    if exists {
        http.Redirect(w, r, "/home", http.StatusSeeOther)  // redirect to home if the user is already logged in
        return
    }
    http.ServeFile(w, r, "../../web/welcome.html")
}

func home(w http.ResponseWriter, r *http.Request) {
    clearCache(w)
    exists, cookie := getCookie(r)
    if !exists {
        http.Redirect(w, r, "/welcome", http.StatusSeeOther)
        return
    }

    conn, err := net.Dial("tcp","127.0.0.1:5000")
    if err != nil {
        fmt.Println("error connecting to port 5000", err)
        http.Redirect(w,r, "/error", http.StatusSeeOther)
        return
    }
    defer conn.Close()

    binary.Write(conn, binary.LittleEndian, GetChirps)
    encoder := gob.NewEncoder(conn)
    encoder.Encode(cookie.Value)

    decoder := gob.NewDecoder(conn)
    var posts []Post
    decoder.Decode(&posts)

    t, err := template.ParseFiles("../../web/homepage.html")
    if err != nil {
        fmt.Println(err)
    }
    t.Execute(w, struct {
        Username string
        Posts []Post
    }{
        cookie.Value,
        posts,
    })
}

func signup(w http.ResponseWriter, r *http.Request) {
    clearCache(w)
    exists, _ := getCookie(r)
    if exists {
        http.Redirect(w, r, "/home", http.StatusSeeOther)
        return
    }
    if r.Method == http.MethodGet {
        http.ServeFile(w, r, "../../web/signup.html")
    } else if r.Method == http.MethodPost {
        conn, err := net.Dial("tcp","127.0.0.1:5000")
        if err != nil {
            fmt.Println("error connecting to port 5000", err)
            http.Redirect(w,r, "/error", http.StatusSeeOther)
            return
        }
        defer conn.Close()

        r.ParseForm()
        if r.PostFormValue("password") != r.PostFormValue("confirm") {
            http.Redirect(w, r, "/signup", http.StatusSeeOther)
            return
        }

        binary.Write(conn, binary.LittleEndian, Signup)
        encoder := gob.NewEncoder(conn)
        encoder.Encode(struct{
            Username string
            Password string
        }{
            r.PostFormValue("username"),
            r.PostFormValue("password"),
        })

        // NOTE: expecting backend to return false if username is taken
        var success bool
        err = binary.Read(conn, binary.LittleEndian, &success)
        if err != nil {
            fmt.Println("error reading from port 5000", err)
            http.Redirect(w,r, "/error", http.StatusSeeOther)
            return
        }

        if !success {
            http.Redirect(w, r, "/signup", http.StatusSeeOther)
            return
        }

        http.SetCookie(w, genCookie(r.PostFormValue("username")))
        http.Redirect(w, r, "/home", http.StatusSeeOther)
        fmt.Printf("Username: %s, Password: %s, Confirmed Pass: %s\n",
            r.PostFormValue("username"),
            r.PostFormValue("password"),
            r.PostFormValue("confirm"))
    }
}

func login(w http.ResponseWriter, r *http.Request) {
    clearCache(w)
    exists, _ := getCookie(r)
    if exists {
        http.Redirect(w, r, "/home", http.StatusSeeOther)
        return
    }
    if r.Method == http.MethodGet {
        http.ServeFile(w, r, "../../web/login.html")
    } else if r.Method == http.MethodPost {
        conn, err := net.Dial("tcp","127.0.0.1:5000")
        if err != nil {
            fmt.Println("error connecting to port 5000", err)
            http.Redirect(w,r, "/error", http.StatusSeeOther)
            return
        }
        defer conn.Close()

        r.ParseForm()
        binary.Write(conn, binary.LittleEndian, Login)
        encoder := gob.NewEncoder(conn)
        encoder.Encode(struct{
            Username string
            Password string
        }{
            r.PostFormValue("username"),
            r.PostFormValue("password"),
        })

        var success bool
        err = binary.Read(conn, binary.LittleEndian, &success)
        if err != nil {
            fmt.Println("error reading from port 5000", err)
            http.Redirect(w,r, "/error", http.StatusSeeOther)
            return
        }

        if !success {
            http.Redirect(w, r, "/login", http.StatusSeeOther)
            return
        }

        http.SetCookie(w, genCookie(r.PostFormValue("username")))
        http.Redirect(w, r, "/home", http.StatusSeeOther)
        fmt.Printf("Username: %s, Password: %s\n",
            r.PostFormValue("username"),
            r.PostFormValue("password"))
    }
}

func logout(w http.ResponseWriter, r *http.Request) {
    clearCache(w)
    cookie, _ := r.Cookie(LOGIN_COOKIE)
    cookie.MaxAge = -1
    cookie.Expires = time.Now().Add(-1 * time.Hour)
    http.SetCookie(w, cookie)
    http.Redirect(w, r, "/welcome", http.StatusSeeOther)
}

func errorPage(w http.ResponseWriter, r *http.Request) {
    clearCache(w)
    t, err := template.ParseFiles("../../web/error.html")
    if err != nil {
        fmt.Println(err)
    }
    t.Execute(w, struct{Username string; Error string}{Username: "Dave", Error: "Singularity"})
}

// the current user (determined by the cookie) will add a new user to their followed list
// based on form value, if follow fails redirect to the error page
func follow(w http.ResponseWriter, r *http.Request) {
    clearCache(w)
    exists, cookie := getCookie(r)
    if !exists {
        http.Redirect(w, r, "/welcome", http.StatusSeeOther)
        return
    }

    if r.Method == http.MethodPost {
        conn, err := net.Dial("tcp","127.0.0.1:5000")
        if err != nil {
            fmt.Println("error connecting to port 5000", err)
            http.Redirect(w,r, "/error", http.StatusSeeOther)
            return
        }
        defer conn.Close()

        r.ParseForm()
        binary.Write(conn, binary.LittleEndian, Follow)
        encoder := gob.NewEncoder(conn)
        encoder.Encode(struct {
            Username1 string
            Username2 string
        }{
            cookie.Value,
            r.PostFormValue("username"),
        })

        var success bool
        err = binary.Read(conn, binary.LittleEndian, &success)
        if err != nil {
            fmt.Println("error reading from port 5000", err)
            http.Redirect(w,r, "/error", http.StatusSeeOther)
            return
        }

        if success {
            http.Redirect(w, r, "/home", http.StatusSeeOther)
        } else {
            http.Redirect(w, r, "/error", http.StatusSeeOther)
        }
    }
}

// reverse logic of follow
func unfollow(w http.ResponseWriter, r *http.Request) {
    clearCache(w)
    exists, cookie := getCookie(r)
    if !exists {
        http.Redirect(w, r, "/welcome", http.StatusSeeOther)
        return
    }

    if r.Method == http.MethodPost {
        conn, err := net.Dial("tcp","127.0.0.1:5000")
        if err != nil {
            fmt.Println("error connecting to port 5000", err)
            http.Redirect(w,r, "/error", http.StatusSeeOther)
            return
        }
        defer conn.Close()

        r.ParseForm()
        binary.Write(conn, binary.LittleEndian, Unfollow)
        encoder := gob.NewEncoder(conn)
        encoder.Encode(struct {
            Username1 string
            Username2 string
        }{
            cookie.Value,
            r.PostFormValue("username"),
        })

        var success bool
        err = binary.Read(conn, binary.LittleEndian, &success)
        if err != nil {
            fmt.Println("error reading from port 5000", err)
            http.Redirect(w,r, "/error", http.StatusSeeOther)
            return
        }

        if success {
            http.Redirect(w, r, "/home", http.StatusSeeOther)
        } else {
            http.Redirect(w, r, "/error", http.StatusSeeOther)
        }
    }

}

// reads a post from form input, then appends it to the slice of posts per user
func submitPost(w http.ResponseWriter, r *http.Request) {
    clearCache(w)
    exists, cookie := getCookie(r)

    conn, err := net.Dial("tcp","127.0.0.1:5000")
    if err != nil {
        fmt.Println("error connecting to port 5000", err)
        http.Redirect(w,r, "/error", http.StatusSeeOther)
        return
    }
    defer conn.Close()

    if !exists {  // modify (also include if there is a cookie that no username is associated with
        http.Redirect(w, r, "/welcome", http.StatusSeeOther)
        return
    }
    if r.Method == http.MethodPost {
        conn, err := net.Dial("tcp","127.0.0.1:5000")
        if err != nil {
            fmt.Println("error connecting to port 5000", err)
            http.Redirect(w,r, "/error", http.StatusSeeOther)
            return
        }
        defer conn.Close()

        r.ParseForm()
        binary.Write(conn, binary.LittleEndian, Chirp)
        encoder := gob.NewEncoder(conn)
        encoder.Encode(struct{
            Username string
            Post string
        }{
            cookie.Value,
            r.PostFormValue("post"),
        })

        var success bool
        err = binary.Read(conn, binary.LittleEndian, &success)
        if err != nil {
            fmt.Println("error reading from port 5000", err)
            http.Redirect(w,r, "/error", http.StatusSeeOther)
            return
        }

        if success {
            http.Redirect(w, r, "/home", http.StatusSeeOther)
        } else {
            http.Redirect(w, r, "/error", http.StatusSeeOther)
        }
    }
}

// searches for a user, provides user info if the user did not search for him/herself
// provides a link to follow/unfollow based on current follow status
func searchResponse(w http.ResponseWriter, r *http.Request) {
    clearCache(w)
    exists, cookie := getCookie(r)
    if !exists {
        http.Redirect(w, r, "/welcome", http.StatusSeeOther)
        return
    }
    if r.Method == http.MethodGet{
        conn, err := net.Dial("tcp","127.0.0.1:5000")
        if err != nil {
            fmt.Println("error connecting to port 5000", err)
            http.Redirect(w,r, "/error", http.StatusSeeOther)
            return
        }
        defer conn.Close()

        r.ParseForm()
        binary.Write(conn, binary.LittleEndian, Search)
        encoder := gob.NewEncoder(conn)
        encoder.Encode(struct{Username string}{r.FormValue("username")})

        var user struct{Follow string}
        decoder := gob.NewDecoder(conn)
        decoder.Decode(&user)

        if r.FormValue("username") != cookie.Value {  // backend function call
            t, _ := template.ParseFiles("../../web/searchResult.html")
            t.Execute(w, struct{Username, Follow string}{r.FormValue("username"), user.Follow})
        } else {
            http.Redirect(w, r, "/home", http.StatusSeeOther)
        }
    }
}

// change deletion to not store nil values
func deleteAccount(w http.ResponseWriter, r *http.Request) {
    clearCache(w)
    cookie, _ := r.Cookie(LOGIN_COOKIE)

    conn, err := net.Dial("tcp","127.0.0.1:5000")
    if err != nil {
        fmt.Println("error connecting to port 5000", err)
        http.Redirect(w,r, "/error", http.StatusSeeOther)
        return
    }
    defer conn.Close()

    binary.Write(conn, binary.LittleEndian, DeleteAccount)
    encoder := gob.NewEncoder(conn)
    encoder.Encode(struct{Username string}{cookie.Value})

    cookie.MaxAge = -1
    cookie.Expires = time.Now().Add(-1 * time.Hour)
    http.SetCookie(w, cookie)
    http.Redirect(w, r, "/welcome", http.StatusSeeOther)
}

func getCookie(r *http.Request) (LoggedIn bool, Cookie *http.Cookie) {
    // ignoring error value because it is likely that the cookie might not exist here
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

func clearCache(w http.ResponseWriter) {
    w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
    w.Header().Set("Pragma", "no-cache")
    w.Header().Set("Expires", "0")
}
