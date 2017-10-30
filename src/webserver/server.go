package main

import(
    . "../../lib"
    "crypto/sha512"
    "encoding/gob"
    "encoding/hex"
    "html/template"
    "log"
    "net"
    "net/http"
    "time"
)

const LOGIN_COOKIE = "loginCookie"  // Cookie to keep users logged in
var LOG map[int]*log.Logger

func main() {
    LOG = InitLog("../../log/frontend.txt")  // create logger map associated with different log codes
    http.HandleFunc("/", welcomeRedirect)  // function for server address page
    http.HandleFunc("/welcome", welcome)   // function for welcome page (main page for not logged in users)
    http.HandleFunc("/signup", signup)     // function for signup page
    http.HandleFunc("/login", login)       // function for login page
    http.HandleFunc("/logout", logout)     // function for logout page
    http.HandleFunc("/home", home)         // function for home page (main page for logged in users)
    http.HandleFunc("/error", errorPage)   // function for error page
    http.HandleFunc("/search-result", searchResult)    // function for search submission
    http.HandleFunc("/delete-account", deleteAccount)  // function for account deletion submission

    gob.Register([]Post{})
    gob.Register(struct{Username, Password string}{})
    gob.Register(struct{Username1, Username2 string}{})
    gob.Register(struct{Searcher, Target string}{})
    gob.Register(struct{Username, Post string}{})

    http.ListenAndServe(":8080", nil)
}

func welcomeRedirect(w http.ResponseWriter, r *http.Request) {
    http.Redirect(w, r, "/welcome", http.StatusSeeOther)  // URL always displays welcome
}

func welcome(w http.ResponseWriter, r *http.Request) {
    LOG[INFO].Println("Welcome Page")
    clearCache(w)
    exists, _ := getCookie(r)
    if exists {
        http.Redirect(w, r, "/home", http.StatusSeeOther)  // redirect to home if the user is already logged in
        return
    }
    http.ServeFile(w, r, "../../web/welcome.html")
}

func home(w http.ResponseWriter, r *http.Request) {
    LOG[INFO].Println("Home Page")
    clearCache(w)
    exists, cookie := getCookie(r)
    if !exists {
        http.Redirect(w, r, "/welcome", http.StatusSeeOther)
        return
    }

    if r.Method == http.MethodGet {
        conn, err := net.Dial("tcp", "127.0.0.1:5000")
        if err != nil {
            LOG[ERROR].Println(StatusText(StatusConnectionError), err)
            http.Redirect(w, r, "/error", http.StatusSeeOther)
            return
        }
        defer conn.Close()

        encoder := gob.NewEncoder(conn)
        err = encoder.Encode(CommandRequest{CommandGetChirps, cookie.Value})
        if err != nil {
            LOG[ERROR].Println(StatusText(StatusEncodeError), err)
            http.Redirect(w, r, "/error", http.StatusSeeOther)
            return
        }

        var response CommandResponse
        decoder := gob.NewDecoder(conn)
        err = decoder.Decode(&response)
        if err != nil {
            LOG[ERROR].Println(StatusText(StatusDecodeError), err)
            http.Redirect(w, r, "/error", http.StatusSeeOther)
            return
        }
        if !response.Success {
            LOG[WARNING].Println(StatusText(response.Status))
            http.Redirect(w, r, "/error", http.StatusSeeOther)
            return
        }

        t, err := template.ParseFiles("../../web/homepage.html")
        if err != nil {
            LOG[ERROR].Println("HTML Template Error", err)
            http.Redirect(w, r, "/error", http.StatusSeeOther)
            return
        }
        err = t.Execute(w, struct {
            Username string
            Posts    interface{}
        }{
            cookie.Value,
            response.Data,
        })
        if err != nil {
            LOG[ERROR].Println("HTML Template Execution Error", err)
            http.Redirect(w, r, "/error", http.StatusSeeOther)
        }
    } else if r.Method == http.MethodPost {
        LOG[INFO].Println("Executing Post")
        conn, err := net.Dial("tcp","127.0.0.1:5000")
        if err != nil {
            LOG[ERROR].Println(StatusText(StatusConnectionError))
            http.Redirect(w, r, "/error", http.StatusSeeOther)
            return
        }
        defer conn.Close()

        r.ParseForm()
        LOG[INFO].Println("Form Values: Post", r.PostFormValue("post"))
        encoder := gob.NewEncoder(conn)
        err = encoder.Encode(CommandRequest{CommandChirp, struct{
            Username string
            Post     string
        }{
            cookie.Value,
            r.PostFormValue("post"),
        }})
        if err != nil {
            LOG[ERROR].Println(StatusText(StatusEncodeError), err)
            http.Redirect(w, r, "/error", http.StatusSeeOther)
            return
        }

        var response CommandResponse
        decoder := gob.NewDecoder(conn)
        err = decoder.Decode(&response)
        if err != nil {
            LOG[ERROR].Println(StatusText(StatusDecodeError))
            http.Redirect(w,r, "/error", http.StatusSeeOther)
            return
        }

        if !response.Success {
            http.Redirect(w, r, "/error", http.StatusSeeOther)
        }
        http.Redirect(w, r, "/home", http.StatusSeeOther)
        LOG[INFO].Println("Post Successfully Submitted")
    }
}

func signup(w http.ResponseWriter, r *http.Request) {
    clearCache(w)
    exists, _ := getCookie(r)
    if exists {
        http.Redirect(w, r, "/home", http.StatusSeeOther)
        return
    }

    if r.Method == http.MethodGet {
        LOG[INFO].Println("Signup Page")
        http.ServeFile(w, r, "../../web/signup.html")
    } else if r.Method == http.MethodPost {
        LOG[INFO].Println("Executing Signup")
        conn, err := net.Dial("tcp","127.0.0.1:5000")
        if err != nil {
            LOG[ERROR].Println(StatusText(StatusConnectionError), err)
            http.Redirect(w,r, "/error", http.StatusSeeOther)
            return
        }
        defer conn.Close()

        err = r.ParseForm()
        if err != nil {
            LOG[ERROR].Println("Form Error", err)
            http.Redirect(w, r, "/error", http.StatusSeeOther)
        }
        LOG[INFO].Println("Form Values: Username", r.PostFormValue("username") + ", Password",
            r.PostFormValue("password") + ",", "Confirm", r.PostFormValue("confirm"))
        if r.PostFormValue("password") != r.PostFormValue("confirm") {
            LOG[INFO].Println("Password Mismatch")
            http.Redirect(w, r, "/signup", http.StatusSeeOther)
            return
        }

        passhash := sha512.Sum512([]byte(r.PostFormValue("password")))
        LOG[INFO].Println("Hex Encoded Passhash", hex.EncodeToString(passhash[:]))
        encoder := gob.NewEncoder(conn)
        err = encoder.Encode(CommandRequest{CommandSignup, struct{
            Username string
            Password string
        }{
            r.PostFormValue("username"),
            hex.EncodeToString(passhash[:]),
        }})
        if err != nil {
            LOG[ERROR].Println(StatusText(StatusEncodeError), err)
            http.Redirect(w, r, "/error", http.StatusSeeOther)
            return
        }

        var response CommandResponse
        decoder := gob.NewDecoder(conn)
        err = decoder.Decode(&response)
        if err != nil {
            LOG[ERROR].Println(StatusText(StatusDecodeError), err)
            http.Redirect(w, r, "/error", http.StatusSeeOther)
            return
        }
        if !response.Success {
            LOG[WARNING].Println(StatusText(response.Status))
            http.Redirect(w, r, "/signup", http.StatusSeeOther)
            return
        }

        LOG[INFO].Println("Successfully Signed Up")
        http.SetCookie(w, genCookie(r.PostFormValue("username")))
        http.Redirect(w, r, "/home", http.StatusSeeOther)
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
        LOG[INFO].Println("Login Page")
        http.ServeFile(w, r, "../../web/login.html")
    } else if r.Method == http.MethodPost {
        LOG[INFO].Println("Executing Login")
        conn, err := net.Dial("tcp","127.0.0.1:5000")
        if err != nil {
            LOG[ERROR].Println(StatusText(StatusConnectionError), err)
            http.Redirect(w, r, "/error", http.StatusSeeOther)
            return
        }
        defer conn.Close()

        r.ParseForm()
        LOG[INFO].Println("Form Values: Username", r.PostFormValue("username") + ", Password",
            r.PostFormValue("password"))
        passhash := sha512.Sum512([]byte(r.PostFormValue("password")))
        LOG[INFO].Println("Hex Encoded Passhash:", hex.EncodeToString(passhash[:]))
        encoder := gob.NewEncoder(conn)
        err = encoder.Encode(CommandRequest{CommandLogin, struct{
            Username string
            Password string
        }{
            r.PostFormValue("username"),
            hex.EncodeToString(passhash[:]),
        }})
        if err != nil {
            LOG[ERROR].Println(StatusText(StatusEncodeError), err)
            http.Redirect(w, r, "/error", http.StatusSeeOther)
            return
        }

        var response CommandResponse
        decoder := gob.NewDecoder(conn)
        err = decoder.Decode(&response)
        if err != nil {
            LOG[ERROR].Println(StatusText(StatusDecodeError), err)
            http.Redirect(w, r, "/error", http.StatusSeeOther)
            return
        }
        if !response.Success {
            LOG[WARNING].Println(StatusText(response.Status))
            http.Redirect(w, r, "/login", http.StatusSeeOther)
            return
        }

        LOG[INFO].Println("Successfully Logged In")
        http.SetCookie(w, genCookie(r.PostFormValue("username")))
        http.Redirect(w, r, "/home", http.StatusSeeOther)
    }
}

func logout(w http.ResponseWriter, r *http.Request) {
    LOG[INFO].Println("Executing Logout")
    clearCache(w)
    cookie, _ := r.Cookie(LOGIN_COOKIE)
    cookie.MaxAge = -1
    cookie.Expires = time.Now().Add(-1 * time.Hour)
    http.SetCookie(w, cookie)
    LOG[INFO].Println("Successfully Logged Out")
    http.Redirect(w, r, "/welcome", http.StatusSeeOther)
}

func errorPage(w http.ResponseWriter, r *http.Request) {
    clearCache(w)
    t, err := template.ParseFiles("../../web/error.html")
    if err != nil {
        LOG[ERROR].Println("HTML Template Error", err)
    }
    err = t.Execute(w, struct{Username string; Error string}{Username: "Dave", Error: "Singularity"})
    if err != nil {
        LOG[ERROR].Println("HTML Template Execution Error", err)
    }
}

// searches for a user, provides user info if the user did not search for him/herself
// provides a link to follow/unfollow based on current follow status
func searchResult(w http.ResponseWriter, r *http.Request) {
    clearCache(w)
    exists, cookie := getCookie(r)
    if !exists {
        http.Redirect(w, r, "/welcome", http.StatusSeeOther)
        return
    }
    if r.Method == http.MethodGet {
        LOG[INFO].Println("Search Results Page")
        conn, err := net.Dial("tcp","127.0.0.1:5000")
        if err != nil {
            LOG[ERROR].Println(StatusText(StatusConnectionError), err)
            http.Redirect(w, r, "/error", http.StatusSeeOther)
            return
        }
        defer conn.Close()

        r.ParseForm()
        LOG[INFO].Println("Form Values: Username", r.PostFormValue("username"))
        if cookie.Value == r.PostFormValue("username") {
            LOG[INFO].Println("User Self Search")
            http.Redirect(w, r, "/home", http.StatusSeeOther)
            return
        }
        encoder := gob.NewEncoder(conn)
        err = encoder.Encode(CommandRequest{CommandSearch, struct{
            Searcher string
            Target   string
        }{
            cookie.Value,
            r.FormValue("username"),
        }})
        if err != nil {
            LOG[ERROR].Println(StatusText(StatusEncodeError), err)
            http.Redirect(w, r, "/error", http.StatusSeeOther)
            return
        }

        var response CommandResponse
        decoder := gob.NewDecoder(conn)
        err = decoder.Decode(&response)
        if err != nil {
            LOG[ERROR].Println(StatusText(StatusDecodeError), err)
            http.Redirect(w, r, "/error", http.StatusSeeOther)
            return
        }
        if !response.Success {
            if response.Status == StatusUserNotFound {
                LOG[WARNING].Println(StatusText(response.Status))
                http.Redirect(w, r, "/home", http.StatusSeeOther)
            } else {
                LOG[ERROR].Println(StatusText(response.Status))
                http.Redirect(w, r, "/error", http.StatusSeeOther)
            }
            return
        }

        t, err := template.ParseFiles("../../web/search-result.html")
        if err != nil {
            LOG[ERROR].Println("HTML Template Error", err)
            http.Redirect(w, r, "/error", http.StatusSeeOther)
            return
        }
        err = t.Execute(w, struct{Username, Follow string}{r.FormValue("username"), response.Data.(string)})
        if err != nil {
            LOG[ERROR].Println("HTML Template Execution Error", err)
            http.Redirect(w, r, "/error", http.StatusSeeOther)
        }
    } else if r.Method == http.MethodPost {
        LOG[INFO].Println("Executing Follow")
        conn, err := net.Dial("tcp","127.0.0.1:5000")
        if err != nil {
            LOG[ERROR].Println(StatusText(StatusConnectionError), err)
            http.Redirect(w, r, "/error", http.StatusSeeOther)
            return
        }
        defer conn.Close()

        r.ParseForm()
        LOG[INFO].Println("Form Values: Username", r.PostFormValue("username"))
        encoder := gob.NewEncoder(conn)
        err = encoder.Encode(CommandRequest{CommandFollow, struct{
            Username1 string
            Username2 string
        }{
            cookie.Value,
            r.PostFormValue("username"),
        }})
        if err != nil {
            LOG[ERROR].Println(StatusText(StatusEncodeError), err)
            http.Redirect(w, r, "/error", http.StatusSeeOther)
            return
        }

        var response CommandResponse
        decoder := gob.NewDecoder(conn)
        err = decoder.Decode(&response)
        if err != nil {
            LOG[ERROR].Println(StatusText(StatusDecodeError))
            http.Redirect(w, r, "/error", http.StatusSeeOther)
            return
        }
        if !response.Success {
            http.Redirect(w, r, "/error", http.StatusSeeOther)  // change
        }
        LOG[INFO].Println("Follow Successful")
        http.Redirect(w, r, "/home", http.StatusSeeOther)
    }
}

// change deletion to not store nil values
func deleteAccount(w http.ResponseWriter, r *http.Request) {
    clearCache(w)
    cookie, _ := r.Cookie(LOGIN_COOKIE)

    conn, err := net.Dial("tcp","127.0.0.1:5000")
    if err != nil {
        LOG[ERROR].Println(StatusText(StatusConnectionError))
        http.Redirect(w,r, "/error", http.StatusSeeOther)
        return
    }
    defer conn.Close()

    encoder := gob.NewEncoder(conn)
    err = encoder.Encode(CommandRequest{CommandDeleteAccount,cookie.Value})
    if err != nil {
        LOG[ERROR].Println(StatusText(StatusEncodeError), err)
        http.Redirect(w, r, "/error", http.StatusSeeOther)
        return
    }

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
