package main

import (
    . "../../lib"
    "bufio"
    "fmt"
    "io/ioutil"
    "net"
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
        bufReader := bufio.NewReader(conn)
        command, err := bufReader.ReadBytes('\n')
        if err != nil {
            fmt.Println("error reading command string ", command)
        }
        runCommand(string(command), conn)
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
            dec := gob.NewDecoder(file)
            var uInfo UserInfo
            err = dec.Decode(&uInfo)
            if err != nil {
                fmt.Println("error decoding, ", err)
                panic(err)
            }
            USERS[uInfo.Username] = &uInfo
        }
    }
}

func runCommand(command string, conn net.Conn){
    switch command {
        case "getChrips": //username

        case "follow":    //username1, username2

        case "unfollow":  //username1, username2

        case "deleteAccount": //username

        case "post":      //username, post

        case "signup":   //username, password

        case "login":    //username, password

        case "search":   //username

        default:
            fmt.Println("Invalid command ", command, ", ignoring.")
    }
}
