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
        defer conn.Close()
        var command int
        err = binary.Read(conn, binary.LittleEndian, &command)
        if err != nil {
            fmt.Println("error reading command ", err)
        }
        runCommand(command, conn)
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
            file, err := os.Open("../../data/"+ user.Name())
            if err != nil {
                fmt.Println("Unable to open file: ", user.Name(),", skipping. ",err)
                continue
            }
            defer file.Close()
            decoder := gob.NewDecoder(file)
            var uInfo UserInfo
            err = decoder.Decode(&uInfo)
            if err != nil {
                fmt.Println("error decoding, ", err)
                panic(err)
            }
            USERS[uInfo.Username] = &uInfo
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
            var signup struct{Username, Password string}
            decoder.Decode(&signup)
            if _, err := os.Stat("../../data/"+signup.Username); !os.IsNotExist(err) {
                binary.Write(conn, binary.LittleEndian, false)
                return
            }
            binary.Write(conn, binary.LittleEndian, true)
        case Login:    //username, password

        case Search:   //username

        default:
            fmt.Println("Invalid command ", command, ", ignoring.")
    }
}
