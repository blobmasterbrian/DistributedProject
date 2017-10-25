package main

import (
    . "../../lib"
    "bufio"
    "fmt"
    "net"
)

var USERS = map[string]*UserInfo{}      // Map of all users

func main(){
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

func loadUsers(

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
