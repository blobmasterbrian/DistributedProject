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
        var command int
        err = binary.Read(conn, binary.LittleEndian, &command)
        if err != nil {
            fmt.Println("error reading command ", err)
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

func runCommand(command int, conn net.Conn){
    decoder := gob.NewDecoder(conn)
    //response := gob.NewEncoder(conn)
    switch command {
        case GetChirps: //username

        case Follow:    //username1, username2

        case Unfollow:  //username1, username2

        case DeleteAccount: //username

        case Chirp:      //username, post

        case Signup:   //username, password
            binary.Write(conn, binary.LittleEndian, signup(decoder))
        case Login:    //username, password
            binary.Write(conn, binary.LittleEndian, login(decoder))
        case Search:   //username

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
    return ok && user.Password == userAndPass.Password
}
