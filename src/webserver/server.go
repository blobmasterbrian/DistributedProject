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
    "os"
    "time"
)

const LOGIN_COOKIE = "loginCookie"  // Cookie to keep users logged in
const ERROR_COOKIE = "errorCookie"   // Cookie to retain error information for error length
var LOG map[int]*log.Logger

func main() {
    if _, err := os.Stat("../../log"); os.IsNotExist(err) {
        os.Mkdir("../../log", os.ModePerm)
    }
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
    exists, _ := getCookie(r, LOGIN_COOKIE)
    if exists {
        http.Redirect(w, r, "/home", http.StatusSeeOther)  // redirect to home if the user is already logged in
        return
    }
    http.ServeFile(w, r, "../../web/welcome.html")
}

func home(w http.ResponseWriter, r *http.Request) {
    LOG[INFO].Println("Home Page")
    clearCache(w)
    exists, cookie := getCookie(r, LOGIN_COOKIE)
    if !exists {
        http.Redirect(w, r, "/welcome", http.StatusSeeOther)
        return
    }

    if r.Method == http.MethodGet {
        response := sendCommand(CommandRequest{CommandGetChirps, cookie.Value})
        if response == nil {
            http.SetCookie(w, genCookie(ERROR_COOKIE, "Send Command Error"))
            http.Redirect(w, r, "/error", http.StatusSeeOther)
            return
        }
        if !response.Success {
            LOG[WARNING].Println(StatusText(response.Status))
            http.SetCookie(w, genCookie(ERROR_COOKIE, StatusText(response.Status)))
            http.Redirect(w, r, "/error", http.StatusSeeOther)
            return
        }

        t, err := template.ParseFiles("../../web/homepage.html")
        if err != nil {
            LOG[ERROR].Println("HTML Template Error", err)
            http.SetCookie(w, genCookie(ERROR_COOKIE, "HTML Template Error"))
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
            http.SetCookie(w, genCookie(ERROR_COOKIE, "HTML Template Execution Error"))
            http.Redirect(w, r, "/error", http.StatusSeeOther)
        }
    } else if r.Method == http.MethodPost {
        LOG[INFO].Println("Executing Post")
        r.ParseForm()
        LOG[INFO].Println("Form Values: Post", r.PostFormValue("post"))
        response := sendCommand(CommandRequest{CommandChirp, struct{
            Username string
            Post     string
        }{
            cookie.Value,
            r.PostFormValue("post"),
        }})
        if response == nil {
            http.SetCookie(w, genCookie(ERROR_COOKIE, "Send Command Error"))
            http.Redirect(w, r, "/error", http.StatusSeeOther)
            return
        }

        if !response.Success {
            http.SetCookie(w, genCookie(ERROR_COOKIE, StatusText(response.Status)))
            http.Redirect(w, r, "/error", http.StatusSeeOther)
        }
        http.Redirect(w, r, "/home", http.StatusSeeOther)
        LOG[INFO].Println("Post Successfully Submitted")
    }
}

func signup(w http.ResponseWriter, r *http.Request) {
    clearCache(w)
    exists, _ := getCookie(r, LOGIN_COOKIE)
    if exists {
        http.Redirect(w, r, "/home", http.StatusSeeOther)
        return
    }

    if r.Method == http.MethodGet {
        LOG[INFO].Println("Signup Page")
        http.ServeFile(w, r, "../../web/signup.html")
    } else if r.Method == http.MethodPost {
        LOG[INFO].Println("Executing Signup")

        err := r.ParseForm()
        if err != nil {
            LOG[ERROR].Println("Form Error", err)
            http.SetCookie(w, genCookie(ERROR_COOKIE, "Form Error"))
            http.Redirect(w, r, "/error", http.StatusSeeOther)
        }

        LOG[INFO].Println("Form Values: Username", r.PostFormValue("username"))
        if r.PostFormValue("password") != r.PostFormValue("confirm") {
            LOG[INFO].Println("Password Mismatch")
            http.Redirect(w, r, "/signup", http.StatusSeeOther)
            return
        }
        if len(r.PostFormValue("username")) == 0 || len(r.PostFormValue("password")) > 0 {
            LOG[INFO].Println("bad param length on signup")
            http.Redirect(w, r, "/signup", http.StatusSeeOther)
            return
        }

        passhash := sha512.Sum512([]byte(r.PostFormValue("password")))
        LOG[INFO].Println("Hex Encoded Passhash", hex.EncodeToString(passhash[:]))
        response := sendCommand(CommandRequest{CommandSignup, struct{
            Username string
            Password string
        }{
            r.PostFormValue("username"),
            hex.EncodeToString(passhash[:]),
        }})
        if response == nil {
            http.SetCookie(w, genCookie(ERROR_COOKIE, "Send Command Error"))
            http.Redirect(w, r, "/error", http.StatusSeeOther)
            return
        }
        if !response.Success {
            LOG[WARNING].Println(StatusText(response.Status))
            http.Redirect(w, r, "/signup", http.StatusSeeOther)
            return
        }

        LOG[INFO].Println("Successfully Signed Up")
        http.SetCookie(w, genCookie(LOGIN_COOKIE, r.PostFormValue("username")))
        http.Redirect(w, r, "/home", http.StatusSeeOther)
    }
}

func login(w http.ResponseWriter, r *http.Request) {
    clearCache(w)
    exists, _ := getCookie(r, LOGIN_COOKIE)
    if exists {
        http.Redirect(w, r, "/home", http.StatusSeeOther)
        return
    }
    if r.Method == http.MethodGet {
        LOG[INFO].Println("Login Page")
        http.ServeFile(w, r, "../../web/login.html")
    } else if r.Method == http.MethodPost {
        LOG[INFO].Println("Executing Login")
        r.ParseForm()
        LOG[INFO].Println("Form Values: Username", r.PostFormValue("username"))
        passhash := sha512.Sum512([]byte(r.PostFormValue("password")))
        LOG[INFO].Println("Hex Encoded Passhash:", hex.EncodeToString(passhash[:]))
        response := sendCommand(CommandRequest{CommandLogin, struct{
            Username string
            Password string
        }{
            r.PostFormValue("username"),
            hex.EncodeToString(passhash[:]),
        }})
        if response == nil {
            http.SetCookie(w, genCookie(ERROR_COOKIE, "Send Command Error"))
            http.Redirect(w, r, "/error", http.StatusSeeOther)
            return
        }
        if !response.Success {
            LOG[WARNING].Println(StatusText(response.Status))
            http.Redirect(w, r, "/login", http.StatusSeeOther)
            return
        }

        LOG[INFO].Println("Successfully Logged In")
        http.SetCookie(w, genCookie(LOGIN_COOKIE, r.PostFormValue("username")))
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
    username := "Dave"
    exists, loginCookie := getCookie(r, LOGIN_COOKIE)
    if exists {
        username = loginCookie.Value
    }
    _, ErrCookie := getCookie(r, ERROR_COOKIE)
    err = t.Execute(w, struct{Username string; Error string}{Username: username, Error: ErrCookie.Value})
    if err != nil {
        LOG[ERROR].Println("HTML Template Execution Error", err)
    }
}

// searches for a user, provides user info if the user did not search for him/herself
// provides a link to follow/unfollow based on current follow status
func searchResult(w http.ResponseWriter, r *http.Request) {
    clearCache(w)
    exists, cookie := getCookie(r, LOGIN_COOKIE)
    if !exists {
        http.Redirect(w, r, "/welcome", http.StatusSeeOther)
        return
    }
    r.ParseForm()
    LOG[INFO].Println("Form Values: Username", r.PostFormValue("username"))
    if cookie.Value == r.PostFormValue("username") {
        LOG[INFO].Println("User Self Search")
        http.Redirect(w, r, "/home", http.StatusSeeOther)
        return
    }
    response := sendCommand(CommandRequest{CommandSearch, struct{
        Searcher string
        Target   string
    }{
        cookie.Value,
        r.FormValue("username"),
    }})
    if response == nil {
        http.SetCookie(w, genCookie(ERROR_COOKIE, "Send Command Error"))
        http.Redirect(w, r, "/error", http.StatusSeeOther)
        return
    }
    if !response.Success {
        if response.Status == StatusUserNotFound {
            LOG[WARNING].Println(StatusText(response.Status))
            http.Redirect(w, r, "/home", http.StatusSeeOther)
        } else {
            LOG[ERROR].Println(StatusText(response.Status))
            http.SetCookie(w, genCookie(ERROR_COOKIE, StatusText(response.Status)))
            http.Redirect(w, r, "/error", http.StatusSeeOther)
        }
        return
    }

    if r.Method == http.MethodGet {
        LOG[INFO].Println("Search Results Page")
        t, err := template.ParseFiles("../../web/search-result.html")
        if err != nil {
            LOG[ERROR].Println("HTML Template Error", err)
            http.SetCookie(w, genCookie(ERROR_COOKIE, "HTML Template Error"))
            http.Redirect(w, r, "/error", http.StatusSeeOther)
            return
        }
        err = t.Execute(w, struct{Username, Follow string}{r.FormValue("username"), response.Data.(string)})
        if err != nil {
            LOG[ERROR].Println("HTML Template Execution Error", err)
            http.SetCookie(w, genCookie(ERROR_COOKIE, "HTML Template Execution Error"))
            http.Redirect(w, r, "/error", http.StatusSeeOther)
        }
    } else if r.Method == http.MethodPost {
        LOG[INFO].Println("Executing Follow/Unfollow")
        LOG[INFO].Println("Form Values: Username", r.PostFormValue("username"))
        r.ParseForm()
        if response.Data == "Follow" {
            response = sendCommand(CommandRequest{CommandFollow, struct {
                Username1 string
                Username2 string
            }{
                cookie.Value,
                r.PostFormValue("username"),
            }})
        } else if response.Data == "Unfollow" {
            response = sendCommand(CommandRequest{CommandUnfollow, struct {
                Username1 string
                Username2 string
            }{
                cookie.Value,
                r.PostFormValue("username"),
            }})
        }
        if response == nil {
            http.SetCookie(w, genCookie(ERROR_COOKIE, "Send Command Error"))
            http.Redirect(w, r, "/error", http.StatusSeeOther)
            return
        }

        if !response.Success {
            http.SetCookie(w, genCookie(ERROR_COOKIE, StatusText(response.Status)))
            http.Redirect(w, r, "/error", http.StatusSeeOther)
            return
        }
        LOG[INFO].Println("Follow Successful")
        http.Redirect(w, r, "/home", http.StatusSeeOther)
    }
}

// change deletion to not store nil values
func deleteAccount(w http.ResponseWriter, r *http.Request) {
    clearCache(w)
    cookie, _ := r.Cookie(LOGIN_COOKIE)
    sendCommand(CommandRequest{CommandDeleteAccount, cookie.Value})
    cookie.MaxAge = -1
    cookie.Expires = time.Now().Add(-1 * time.Hour)
    http.SetCookie(w, cookie)
    http.Redirect(w, r, "/welcome", http.StatusSeeOther)
}

func getCookie(r *http.Request, cookiename string) (LoggedIn bool, Cookie *http.Cookie) {
    // ignoring error value because it is likely that the cookie might not exist here
    cookie, _ := r.Cookie(cookiename)
    if cookie == nil {
        return false, nil
    }
    return true, cookie
}

func genCookie(cookiename, value string) *http.Cookie {
    return &http.Cookie{
        Name:     cookiename,
        Value:    value,
        Expires:  time.Now().Add(24 * time.Hour),
    }
}

func clearCache(w http.ResponseWriter) {
    w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
    w.Header().Set("Pragma", "no-cache")
    w.Header().Set("Expires", "0")
}

func sendCommand(command CommandRequest) *CommandResponse {
    conn, err := net.Dial("tcp", "127.0.0.1:5000")
    if err != nil {
        LOG[ERROR].Println(StatusText(StatusConnectionError), err)
        return nil
    }
    defer conn.Close()

    encoder := gob.NewEncoder(conn)
    err = encoder.Encode(command)
    if err != nil {
        LOG[ERROR].Println(StatusText(StatusEncodeError), err)
        return nil
    }

    var response CommandResponse
    decoder := gob.NewDecoder(conn)
    err = decoder.Decode(&response)
    if err != nil {
        LOG[ERROR].Println(StatusText(StatusDecodeError), err)
        return nil
    }
    return &response
}
