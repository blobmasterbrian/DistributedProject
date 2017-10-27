package main

import (
    . "../../lib"
    "encoding/gob"
    "fmt"
    "io/ioutil"
    "log"
    "net"
    "os"
)

var USERS = map[string]*UserInfo{}      // Map of all users
var LOG map[string]*log.Logger

func main(){
    LOG = InitLog("../../log/backend.txt")
    loadUsers()

    server, err := net.Listen("tcp", ":5000")
    if err != nil {
        LOG["error"].Println("error starting server ", err)
        return
    }
    for {
        conn, err := server.Accept()
        if err != nil {
            LOG["error"].Println("error accepting connection ", err)
            continue
        }

        var request CommandRequest
        gob.Register(struct {Username, Password string}{})
        decoder := gob.NewDecoder(conn)
        decoder.Decode(&request)
        if err != nil {
            LOG["error"].Println("error reading command ", err)
            continue
        }
        runCommand(conn, request)
        conn.Close()
    }
}


/*
    Load Users reads encoded gob files from the data directory and fills the USERS map with
    the corresponding UserInfo data.
    Each file in the data directory represents a single user, the file is opened, decoded and stored
    in the USERS map.
*/
func loadUsers() {
    users, err := ioutil.ReadDir("../../data")
    if err != nil {
        LOG["error"].Println("unable to read the data directory ", err)
        panic(err)
    }
    for _, user := range users {
        if !user.IsDir() {
            file, err := os.Open("../../data/" + user.Name())
            if err != nil {
                LOG["warning"].Println("Unable to open file ", user.Name(), ", skipping", err)
                continue
            }
            decoder := gob.NewDecoder(file)
            var uInfo UserInfo
            err = decoder.Decode(&uInfo)
            if err != nil {
                LOG["error"].Println("Unable to decode ", err)
                panic(err)
            }
            USERS[uInfo.Username] = &uInfo
            LOG["info"].Println("Load user", uInfo.Username)
            file.Close()
        }
    }
}

func runCommand(conn net.Conn, request CommandRequest){
    LOG["info"].Println("Running command ", request.CommandCode)
    serverEncoder := gob.NewEncoder(conn)
    switch request.CommandCode {
        case CommandSignup:
            signup(serverEncoder, request)
        case CommandDeleteAccount:

        case CommandLogin:
            login(serverEncoder, request)
        case CommandFollow:
            follow(serverEncoder, request)
        case CommandUnfollow:
            unfollow(serverEncoder, request)
        case CommandSearch:
            search(serverEncoder, request)
        case CommandChirp:
            chirp(serverEncoder, request)
        case CommandGetChirps:
            getChrips(serverEncoder, request)
        default:
            LOG["warning"].Println("Invalid command ", request.CommandCode, ", ignoring.")
    }
}

/*
    signup takes in a decoder as an argument with an expected decode resulting in a 
    username password combo.
    signup returns true or false representing whether a user was created sucessfully

    signup then tries to create a file ../../data/*username* and encode a new UserInfo
    object into the file.
    if this is successful, the new UserInfo object is added to the USERS map 
*/
func signup(serverEncoder *gob.Encoder, request CommandRequest) {
    userAndPass, ok := request.Data.(struct{Username, Password string})
    fmt.Println(ok)  // ok should be used for error checking in future

    if _, err := os.Stat("../../data/" + userAndPass.Username); !os.IsNotExist(err) {
       LOG["info"].Println("username", userAndPass.Username, "already exists", err)
       // encode CommandResponse with failed success and proper Status Code
    }

    file, err := os.Create("../../data/" + userAndPass.Username)
    if err != nil {
        LOG["error"].Println("unable to create file ", err)
        // same as above
    }
    defer file.Close()

    fileEncoder := gob.NewEncoder(file)
    newUser :=  NewUserInfo(userAndPass.Username, userAndPass.Password)
    err = fileEncoder.Encode(newUser)
    if err != nil {
        LOG["error"].Println("error encoding new user ", err)
        // same as above
    }
    USERS[newUser.Username] = newUser
    serverEncoder.Encode(CommandResponse{true, StatusAccepted, nil})
}

func login(serverEncoder *gob.Encoder, request CommandRequest) {
    userAndPass := request.Data.(struct{Username, Password string})

    user, ok := USERS[userAndPass.Username]
    if !ok {
        LOG["error"].Println("Could not find ", userAndPass.Username, " in map")
        // same as above
    }
    if user.Password != userAndPass.Password {
        LOG["info"].Println("Password ", user.Password, " did not match ", userAndPass.Password)
        // same as above
    }
    // check condition? return ok && user.Password == userAndPass.Password
    serverEncoder.Encode(CommandResponse{true, StatusAccepted, nil})
 }

func follow(serverEncoder *gob.Encoder, request CommandRequest) {
    users := request.Data.(struct{Username1, Username2 string})

    user, ok := USERS[users.Username1]
    user2, ok2 := USERS[users.Username2]
    if !ok || !ok2 {
        LOG["error"].Println("User does not exist")
        serverEncoder.Encode(CommandResponse{false, StatusUserNotFound, nil})
    }
    user.Follow(user2)  // error check to see if it succeeds
    serverEncoder.Encode(CommandResponse{true, StatusAccepted, nil})
}

func unfollow(serverEncoder *gob.Encoder, request CommandRequest) {
    users := request.Data.(struct{Username1, Username2 string})

    user, ok := USERS[users.Username1]
    user2, ok2 := USERS[users.Username2]
    if !ok || !ok2 {
        LOG["error"].Println("User does not exist")
        serverEncoder.Encode(CommandResponse{false, StatusUserNotFound, nil})
    }
    user.UnFollow(user2)  // same as above
    serverEncoder.Encode(CommandResponse{true, StatusAccepted, nil})
}

func search(serverEncoder *gob.Encoder, request CommandRequest) {
    username := request.Data.(struct{Searcher, Target string})

    user1, ok := USERS[username.Searcher]
    user2, ok2 := USERS[username.Target]
    if !ok || !ok2 {
        serverEncoder.Encode(CommandResponse{false, StatusUserNotFound, nil})
    } else {
        if user1.IsFollowing(user2) {
            serverEncoder.Encode(CommandResponse{true, StatusUserFollowed, "unfollow"})
        } else {
            serverEncoder.Encode(CommandResponse{true, StatusUserNotFollowed, "follow"})
        }
    }
}

func chirp(serverEncoder *gob.Encoder, request CommandRequest) {
    postInfo := request.Data.(struct{Username, Post string})

    user, ok := USERS[postInfo.Username]
    if !ok {
        serverEncoder.Encode(CommandResponse{false, StatusUserNotFound, nil})
        return
    }
    user.WritePost(postInfo.Post)

    file, err := os.Create("../../data/" + user.Username)
    if err != nil {
        LOG["error"].Println("unable to create file ", err)
        // return false: error creating
    }
    defer file.Close()

    encoder := gob.NewEncoder(file)
    err = encoder.Encode(user)
    if err != nil {
        LOG["error"].Println("Unable to encode user ", err)
        serverEncoder.Encode(CommandResponse{false, StatusEncodeError, nil})
    return
    }
    serverEncoder.Encode(CommandResponse{true, StatusAccepted, nil})
}

// We should have docstrings for functions to explain what is expected as the Data
// member of CommandRequest kinda like the comments we had previously in runCommand
func getChrips(serverEncoder *gob.Encoder, request CommandRequest) {
    username := request.Data.(string)

    gob.Register([]Post{})  // register post slice as implementing interface
    user, ok := USERS[username]
    if !ok {
        serverEncoder.Encode(CommandResponse{false, StatusUserNotFound, nil})
        return
    }
    err := serverEncoder.Encode(CommandResponse{true, StatusAccepted, user.GetAllChirps()})
    if err != nil {
        LOG["error"].Println("Unable to encode chirps for user: ", username)
        // encode failure response?
    }
}
