package main

import (
    . "../../lib"
    "encoding/binary"
    "encoding/gob"
    "fmt"
    "io/ioutil"
    "net"
    "os"
)

var USERS = map[string]*UserInfo{}      // Map of all users

func main(){
    loadUsers()

    server, err := net.Listen("tcp", ":5000")
    if err != nil {
        fmt.Println("error starting server ", err)
        return
    }
    for {
        conn, err := server.Accept()
        if err != nil {
            fmt.Println("error accepting connection ", err)
        }
        var command int32
        err = binary.Read(conn, binary.LittleEndian, &command)
        if err != nil {
            fmt.Println("error reading command ", err)
            continue
        }
        runCommand(command, conn)
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
        fmt.Println("unable to open lib directory")
        panic(err)
    }
    for _, user := range users {
        if !user.IsDir() {
            file, err := os.Open("../../data/" + user.Name())
            if err != nil {
                fmt.Println("Unable to open file: ", user.Name(),", skipping. ",err)
                continue
            }
            decoder := gob.NewDecoder(file)
            var uInfo UserInfo
            err = decoder.Decode(&uInfo)
            if err != nil {
                fmt.Println("error decoding, ", err)
                panic(err)
            }
            USERS[uInfo.Username] = &uInfo
            file.Close()
        }
    }
}

func runCommand(command int32, conn net.Conn){
    decoder := gob.NewDecoder(conn)
    response := gob.NewEncoder(conn)
    switch command {
        case Signup:
            binary.Write(conn, binary.LittleEndian, signup(decoder))
        case DeleteAccount:

        case Login:
            binary.Write(conn, binary.LittleEndian, login(decoder))
        case Follow:

        case Unfollow:

        case Search:

        case Chirp:
            binary.Write(conn, binary.LittleEndian, chirp(decoder))
        case GetChirps:
            getChrips(decoder, response)
        default:
            fmt.Println("Invalid command ", command, ", ignoring.")
    }
}

/*
    signup takes in a decoder as an argument with an expected decode resulting in a 
    username password combo.
    signup returns true or false representing wether a user was created sucessfully

    signup then tries to create a file ../../data/*username* and encode a new UserInfo
    object into the file.
    if this is successful, the new UserInfo object is added to the USERS map 
*/
func signup(decoder *gob.Decoder) bool {
    var userAndPass struct{Username, Password string}
    err := decoder.Decode(&userAndPass)
    if err != nil {
        fmt.Println("error decoding ", err)
        return false
    }

    if _, err := os.Stat("../../data/" + userAndPass.Username); !os.IsNotExist(err) {
        return false
    }

    file, err := os.Create("../../data/" + userAndPass.Username)
    if err != nil {
        fmt.Println("unable to create file ", err)
        return false
    }
    defer file.Close()

    encoder := gob.NewEncoder(file)
    newUser :=  NewUserInfo(userAndPass.Username, userAndPass.Password)
    err = encoder.Encode(newUser)
    if err != nil {
        fmt.Println("error encoding new user ", err)
        return false
    }
    USERS[newUser.Username] = newUser
    return true
}

func login(decoder *gob.Decoder) bool {
    var userAndPass struct{Username, Password string}
    err := decoder.Decode(&userAndPass)
    if err != nil {
        fmt.Println("error decoding ", err)
        return false
    }
    user, ok := USERS[userAndPass.Username]
    if !ok {
        fmt.Println("Could not find ", userAndPass.Username, " in map")
    }
    if user.Password != userAndPass.Password {
        fmt.Println("Password ", user.Password, " did not match ", userAndPass.Password)
    }
    return ok && user.Password == userAndPass.Password
}

func chirp(decoder *gob.Decoder) bool {
    var postInfo struct {Username, NewPost string}
    err := decoder.Decode(&postInfo)
    if err != nil {
        fmt.Println("Unable to decode user and post info ", err)
        return false
    }
    user, ok := USERS[postInfo.Username]
    if !ok {
        return false
    }
    user.WritePost(NewPost)
    return true
}

func getChrips(decoder *gob.Decoder, response *gob.Encoder) {
    var username string
    err := decoder.Decode(&username)
    if err != nil {
        fmt.Println("Unable to decode username ", err)
        return
    }
    var result []Post{}
    user, ok := USERS[username]
    if ok {
        result = USERS[username].GetAllChirps()
    }
    err = response.Encode(result)
    if err != nil {
        fmt.Println("Unable to encode chirps for user: ", username)
    }
}
