package main

import (
    . "../../lib"
    "encoding/gob"
    "io/ioutil"
    "log"
    "net"
    "os"
)

var USERS = map[string]*UserInfo{}      // Map of all users
var LOG map[int]*log.Logger

func main() {
    LOG = InitLog("../../log/backend.txt")
    loadUsers()

    server, err := net.Listen("tcp", ":5000")
    if err != nil {
        LOG[ERROR].Println("Error starting server ", err)
        return
    }
    gob.Register([]Post{})
    gob.Register(struct{Username, Password string}{})
    gob.Register(struct{Username1, Username2 string}{})
    gob.Register(struct{Searcher, Target string}{})
    gob.Register(struct{Username, Post string}{})
    for {
        conn, err := server.Accept()
        if err != nil {
            LOG[ERROR].Println(StatusText(StatusConnectionError), err)
            continue
        }

        var request CommandRequest
        decoder := gob.NewDecoder(conn)
        decoder.Decode(&request)
        if err != nil {
            LOG[ERROR].Println(StatusText(StatusDecodeError), err)
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
        LOG[ERROR].Println("Unable to read the data directory", err)
        panic(err)
    }
    for _, user := range users {
        if !user.IsDir() {
            file, err := os.Open("../../data/" + user.Name())
            if err != nil {
                LOG[WARNING].Println("Unable to open file", user.Name(), ", skipping", err)
                continue
            }
            decoder := gob.NewDecoder(file)
            var uInfo UserInfo
            err = decoder.Decode(&uInfo)
            if err != nil {
                LOG[ERROR].Println(StatusText(StatusDecodeError), err)
                file.Close()
                continue
            }
            USERS[uInfo.Username] = &uInfo
            LOG[INFO].Println("Load user", uInfo.Username)
            file.Close()
        }
    }
}

func runCommand(conn net.Conn, request CommandRequest) {
    LOG[INFO].Println("Running command ", request.CommandCode)
    serverEncoder := gob.NewEncoder(conn)
    switch request.CommandCode {
        case CommandSignup:  // map int to function pointer no case switch necessary
            signup(serverEncoder, request)
        case CommandDeleteAccount:
            deleteAccount(serverEncoder, request)
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
            LOG[WARNING].Println("Invalid command ", request.CommandCode, ", ignoring.")
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
    if !ok {
        LOG[ERROR].Println(StatusText(StatusDecodeError))
        serverEncoder.Encode(CommandResponse{false, StatusDecodeError, nil})
        return
    }

    if _, err := os.Stat("../../data/" + userAndPass.Username); !os.IsNotExist(err) {
       LOG[INFO].Println("Username", userAndPass.Username, "already exists", err)
       serverEncoder.Encode(CommandResponse{false, StatusDuplicateUser, nil})
       return
    }


    newUser :=  NewUserInfo(userAndPass.Username, userAndPass.Password)
    writeUser(newUser)

    USERS[newUser.Username] = newUser
    LOG[INFO].Println("Created user", newUser.Username)
    serverEncoder.Encode(CommandResponse{true, StatusAccepted, nil})
}

func deleteAccount(serverEncoder *gob.Encoder, request CommandRequest) {
    username, ok := request.Data.(string)
    if !ok {
        LOG[ERROR].Println(StatusText(StatusDecodeError))
        serverEncoder.Encode(CommandResponse{false, StatusDecodeError, nil})
        return
    }
    user, ok := USERS[username]
    if !ok {
        LOG[INFO].Println(StatusText(StatusUserNotFound), username)
        serverEncoder.Encode(CommandResponse{false, StatusUserNotFound, nil})
        return
    }
    for _, otherUser := range user.FollowedBy {
        USERS[otherUser].UnFollow(user)
    }
    os.Remove("../../data/" + user.Username)
    delete(USERS, user.Username)
    serverEncoder.Encode(CommandResponse{true, StatusAccepted, nil})
}

func login(serverEncoder *gob.Encoder, request CommandRequest) {
    userAndPass, ok := request.Data.(struct{Username, Password string})
    if !ok {
        LOG[ERROR].Println(StatusText(StatusDecodeError))
        serverEncoder.Encode(CommandResponse{false, StatusDecodeError, nil})
        return
    }
    user, ok := USERS[userAndPass.Username]
    if !ok {
        LOG[INFO].Println(StatusText(StatusUserNotFound), userAndPass.Username)
        serverEncoder.Encode(CommandResponse{false, StatusUserNotFound, nil})
        return
    }
    if user.Password != userAndPass.Password {
        LOG[INFO].Println("Password", user.Password, "did not match", userAndPass.Password)
        serverEncoder.Encode(CommandResponse{false, StatusIncorrectPassword, nil})
        return
    }
    LOG[INFO].Println("User", user.Username, "login")
    serverEncoder.Encode(CommandResponse{true, StatusAccepted, nil})
 }

func follow(serverEncoder *gob.Encoder, request CommandRequest) {
    users, ok := request.Data.(struct{Username1, Username2 string})
    if !ok {
        LOG[ERROR].Println(StatusText(StatusDecodeError))
        serverEncoder.Encode(CommandResponse{false, StatusDecodeError, nil})
        return
    }

    user, ok := USERS[users.Username1]
    user2, ok2 := USERS[users.Username2]
    if !ok || !ok2 {
        LOG[WARNING].Println(StatusText(StatusUserNotFound))
        serverEncoder.Encode(CommandResponse{false, StatusUserNotFound, nil})
        return
    }
    if !user.Follow(user2) {
        LOG[ERROR].Println("User", user.Username, "unable to follow", user2.Username)
        serverEncoder.Encode(CommandResponse{false, StatusInternalError, nil})
        return
    }
    writeUser(user)

    serverEncoder.Encode(CommandResponse{true, StatusAccepted, nil})
}

func unfollow(serverEncoder *gob.Encoder, request CommandRequest) {
    users, ok := request.Data.(struct{Username1, Username2 string})
    if !ok {
        LOG[ERROR].Println(StatusText(StatusDecodeError))
        serverEncoder.Encode(CommandResponse{false, StatusDecodeError, nil})
        return
    }

    user, ok := USERS[users.Username1]
    user2, ok2 := USERS[users.Username2]
    if !ok || !ok2 {
        LOG[WARNING].Println(StatusText(StatusUserNotFound))
        serverEncoder.Encode(CommandResponse{false, StatusUserNotFound, nil})
        return
    }
    if !user.UnFollow(user2) {
        LOG[ERROR].Println("User", user.Username, "unable to unfollow", user2.Username)
        serverEncoder.Encode(CommandResponse{false, StatusInternalError, nil})
        return
    }
    writeUser(user)

    serverEncoder.Encode(CommandResponse{true, StatusAccepted, nil})
}

func search(serverEncoder *gob.Encoder, request CommandRequest) {
    username, ok := request.Data.(struct{Searcher, Target string})
    if !ok {
        LOG[ERROR].Println(StatusText(StatusDecodeError))
        serverEncoder.Encode(CommandResponse{false, StatusDecodeError, nil})
        return
    }

    user1, ok := USERS[username.Searcher]
    user2, ok2 := USERS[username.Target]
    if !ok || !ok2 {
        LOG[WARNING].Println(StatusText(StatusUserNotFound))
        serverEncoder.Encode(CommandResponse{false, StatusUserNotFound, nil})  // change to true cause not failure occurred, split ok1 and ok2
    } else {
        LOG[INFO].Println("User", user1.Username, "search", user2.Username)
        if user1.IsFollowing(user2) {
            serverEncoder.Encode(CommandResponse{true, StatusUserFollowed, "Unfollow"})
        } else {
            serverEncoder.Encode(CommandResponse{true, StatusUserNotFollowed, "Follow"})
        }
    }
}

func chirp(serverEncoder *gob.Encoder, request CommandRequest) {
    postInfo, ok := request.Data.(struct{Username, Post string})
    if !ok {
        LOG[ERROR].Println(StatusText(StatusDecodeError))
        serverEncoder.Encode(CommandResponse{false, StatusDecodeError, nil})
        return
    }

    user, ok := USERS[postInfo.Username]
    if !ok {
        LOG[WARNING].Println(StatusText(StatusUserNotFound), postInfo.Username)
        serverEncoder.Encode(CommandResponse{false, StatusUserNotFound, nil})
        return
    }
    user.WritePost(postInfo.Post)
    writeUser(user)

    serverEncoder.Encode(CommandResponse{true, StatusAccepted, nil})
}

// We should have docstrings for functions to explain what is expected as the Data
// member of CommandRequest kinda like the comments we had previously in runCommand
func getChrips(serverEncoder *gob.Encoder, request CommandRequest) {
    username, ok := request.Data.(string)
    if !ok {
        LOG[ERROR].Println(StatusText(StatusDecodeError))
        serverEncoder.Encode(CommandResponse{false, StatusDecodeError, nil})
        return
    }

    user, ok := USERS[username]
    if !ok {
        LOG[WARNING].Println(StatusText(StatusUserNotFound), username)
        serverEncoder.Encode(CommandResponse{false, StatusUserNotFound, nil})
        return
    }
    serverEncoder.Encode(CommandResponse{true, StatusAccepted, user.GetAllChirps()})
}

func writeUser(user *UserInfo) {
    file, err := os.Create("../../data/" + user.Username)
    if err != nil {
        LOG[ERROR].Println("Unable to create file ", err)
        return
    }
    defer file.Close()

    encoder := gob.NewEncoder(file)
    err = encoder.Encode(user)
    if err != nil {
        LOG[ERROR].Println(StatusText(StatusEncodeError), err)
        return
    }

}
