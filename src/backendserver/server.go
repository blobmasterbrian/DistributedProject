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
    if _, err := os.Stat("../../log"); os.IsNotExist(err) {
        os.Mkdir("../../log", os.ModePerm)
    }
    if _, err := os.Stat("../../data"); os.IsNotExist(err) {
        os.Mkdir("../../data", os.ModePerm)
    }
    LOG = InitLog("../../log/backend.txt")
    loadUsers()

    server, err := net.Listen("tcp", ":5000")
    if err != nil {
        LOG[ERROR].Println("Error starting server ", err)
        return
    }
    //register for encoding and decoding struct values within data types
    gob.Register([]Post{})
    gob.Register(struct{Username, Password string}{})
    gob.Register(struct{Username1, Username2 string}{})
    gob.Register(struct{Searcher, Target string}{})
    gob.Register(struct{Username, Post string}{})

    //main loop for accepting and running web server commands
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

//run commmand is a basic switch case statment, running required functions based off
//command codes.  A server encoder is created and passed on to the functions so they can respond.
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

//delete account takes a username, then finds the corresponding user
//It then calls unfollow on the current user and has the current user unfollow
//all users it is currently following to remove dead references
//it then removes the user from the map and deletes the user from 
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
        writeUser(USERS[otherUser])
    }
    for key := range user.Following {
        user.UnFollow(USERS[key])
        writeUser(USERS[key])
    }
    os.Remove("../../data/" + user.Username)
    delete(USERS, user.Username)
    serverEncoder.Encode(CommandResponse{true, StatusAccepted, nil})
}

//login takes a username password combo from the command request
//it then checks these values against the values stored in the map and returns
//relivant success info
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


//follow takes two strings from the command response and then calls follow on the first to the second
//it returns relivant error information if the follow fails or one of the users does not exist
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
        return1111
    }
    if !user.Follow(user2) {
        LOG[ERROR].Println("User", user.Username, "unable to follow", user2.Username)
        serverEncoder.Encode(CommandResponse{false, StatusInternalError, nil})
        return
    }
    writeUser(user)
    writeUser(user2)

    serverEncoder.Encode(CommandResponse{true, StatusAccepted, nil})
}

//follow is similar to above but with reverse functionality, kept as separate functions
//for ease of front end data sending
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
    writeUser(user2)

    serverEncoder.Encode(CommandResponse{true, StatusAccepted, nil})
}


//Search takes a command request with two strings: the searcher username and the target username
//It then performs the specified search and returns if the user is following the target
//it returns relivant error info if one of the users does not exist
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
        serverEncoder.Encode(CommandResponse{false, StatusUserNotFound, nil})
    } else {
        LOG[INFO].Println("User", user1.Username, "search", user2.Username)
        if user1.IsFollowing(user2) {
            serverEncoder.Encode(CommandResponse{true, StatusUserFollowed, "Unfollow"})
        } else {
            serverEncoder.Encode(CommandResponse{true, StatusUserNotFollowed, "Follow"})
        }
    }
}

//Chirp takes a command request with a Username Post string combo and calls the corrisponding
//write Post function for the specified user.  It writes the change to a file and then
//responds with CommandResponse containing corresponding error info
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

//Get chirps takes in a CommandRequest which contains a string that represents the user that
//the frontend is trying to get the chrips of.  the corresponding call to getChrips is called
//and are encoded back to the front end
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
    serverEncoder.Encode(CommandResponse{true, StatusAccepted, user.GetAllChirps(USERS)})
}

//writeUser takes in a user info pointer and writes the user info to a file using gob
//there is no return value but logs are created on error
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
