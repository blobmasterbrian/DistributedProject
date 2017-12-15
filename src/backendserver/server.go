package main

import (
    . "../../lib"
    "encoding/gob"
    "io/ioutil"
    "log"
    "net"
    "os"
    "strconv"
    "sync"
    "time"
)

var USERS_LOCK = &sync.RWMutex{}
var USERS = map[string]*UserInfo{}      // Map of all users
var LOG map[int]*log.Logger

func main() {
    if _, err := os.Stat("../../log"); os.IsNotExist(err) {
        os.Mkdir("../../log", os.ModePerm)
    }
    if _, err := os.Stat("../../data"); os.IsNotExist(err) {
        os.Mkdir("../../data", os.ModePerm)
    }
    LOG = InitLog("../../log/backend.log")

    //register for encoding and decoding struct values within data types
    gob.Register([]Post{})
    gob.Register(struct{Username, Password string}{})
    gob.Register(struct{Username1, Username2 string}{})
    gob.Register(struct{Searcher, Target string}{})
    gob.Register(struct{Username, Post string}{})
    gob.Register(struct{Id int; Serverlist []int}{})

    replica := NewReplica()

    portChannel := make(chan int)
    userChannel := make(chan *UserInfo)
    go replica.DetermineMaster(portChannel, userChannel, &USERS, USERS_LOCK)
    <-portChannel
    if replica.IsMaster {
        loadUsers()
    } else {
        for uInfo := range userChannel {
            writeUser(uInfo)
            USERS[uInfo.Username] = uInfo
        }
    }
    <-portChannel

    addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:" + strconv.Itoa(replica.Port))
    if err != nil {
        LOG[WARNING].Println("TCPAddr struct could not be created", err)
    }
    server, err := net.ListenTCP("tcp", addr)
    if err != nil {
        LOG[WARNING].Println("Unable to listen on master port, rerunning determine master")
        USERS_LOCK.Lock()
        USERS = map[string]*UserInfo{}
        USERS_LOCK.Unlock()
        replica.ResetServers()
        replica.DetermineMaster(portChannel, userChannel, &USERS, USERS_LOCK)
        <-portChannel
        if replica.IsMaster {
            LOG[ERROR].Println("double resolve to master, unable to listen", err)
            panic("Server insists it is the master when it is not")
        }

        addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:" + strconv.Itoa(replica.Port))
        if err != nil {
            LOG[WARNING].Println("TCPAddr struct could not be created", err)
        }
        server, err = net.ListenTCP("tcp", addr)
        if err != nil {
            LOG[ERROR].Println("Unable to listen on port", replica.Port, err)
            return
        }
    }

    //main loop for accepting and running web server commands
    for {
        if !replica.IsMaster {
            server.SetDeadline(time.Now().Add(15 * time.Second))
        }
        conn, err := server.Accept()
        if err != nil {
            nErr := err.(*net.OpError)
            if nErr.Timeout() {
                LOG[WARNING].Println("Master Is Ded, Running Election")
                // election
            }
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
        if replica.IsMaster {
            replica.PropagateRequest(request)
        }
        go runCommand(conn, request)
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
    USERS_LOCK.Lock()
    defer USERS_LOCK.Unlock()
    for _, user := range users {
        if !user.IsDir() {
            file, err := os.Open("../../data/" + user.Name())
            if err != nil {
                LOG[WARNING].Println("Unable to open file", user.Name(), ", skipping", err)
                continue
            }
            decoder := gob.NewDecoder(file)
            uInfo := NewUserInfo("","")
            err = decoder.Decode(uInfo)
            if err != nil {
                LOG[ERROR].Println(StatusText(StatusDecodeError), err)
                file.Close()
                continue
            }
            USERS[uInfo.Username] = uInfo
            LOG[INFO].Println("Load user", uInfo.Username)
            file.Close()
        }
    }
}

//run command is a basic switch case statement, running required functions based off
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
        case CommandSendPing:
            LOG[INFO].Println("Ping Received from Master")
        case CommandConstructFilesystem:
            LOG[WARNING].Println("Filesystem Already Constructed")
        default:
            LOG[WARNING].Println("Invalid command ", request.CommandCode, ", ignoring.")
    }
    conn.Close()
}

/*
    signup takes in a decoder as an argument with an expected decode resulting in a 
    username password combo.
    signup returns true or false representing whether a user was created successfully

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

    USERS_LOCK.Lock()
    USERS[newUser.Username] = newUser
    USERS_LOCK.Unlock()

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

    USERS_LOCK.Lock()
    defer USERS_LOCK.Unlock()
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

    USERS_LOCK.RLock()
    defer USERS_LOCK.RUnlock()
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

    USERS_LOCK.RLock()
    defer USERS_LOCK.RUnlock()
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

    USERS_LOCK.RLock()
    defer USERS_LOCK.RUnlock()
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

    USERS_LOCK.RLock()
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
    USERS_LOCK.RUnlock()
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

    USERS_LOCK.RLock()
    defer USERS_LOCK.RUnlock()
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

    USERS_LOCK.RLock()
    defer USERS_LOCK.RUnlock()
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
    user.Lock()
    defer user.Unlock()
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
