package lib

import (
	"sync"
	"time"
	"net"
	"strconv"
	"encoding/gob"
	"log"
)


type ReplicaInfo struct {
	id            int
	masterId      int
	isMaster      bool
	serverMutex   *sync.Mutex
	activeServers []int
	LOG           map[int]*log.Logger
}

func NewReplica() ReplicaInfo {
	return ReplicaInfo{
		id:            -1,
		masterId:      -1,
		isMaster:      false,
		serverMutex:   &sync.Mutex{},
		activeServers: []int{},
		LOG:           InitLog("../../log/replica.log"),
	}
}

func (replica *ReplicaInfo) ResetServers() {
	replica.serverMutex.Lock()
	replica.activeServers = []int{}
	replica.serverMutex.Unlock()
}

func (replica *ReplicaInfo) sendPings() {
	for {
		time.Sleep(3 * time.Second)
		replica.LOG[INFO].Println("Initiate Pinging")
		for i, port := range replica.activeServers {
			replica.LOG[INFO].Println("Pinging server at port", port)
			replica.serverMutex.Lock()
			conn, err := net.Dial("tcp", ":" + strconv.Itoa(port))
			if err != nil {
				conn, err = net.Dial("tcp", ":" + strconv.Itoa(port))
			}
			if err != nil {
				replica.LOG[WARNING].Println("Server at port", port, "is dead")
				replica.activeServers = append(replica.activeServers[:i], replica.activeServers[i+1:]...)
			}
			replica.serverMutex.Unlock()

			command := CommandRequest{CommandSendPing, nil}
			encoder := gob.NewEncoder(conn)
			err = encoder.Encode(command)
			if err != nil {
				replica.LOG[ERROR].Println(StatusText(StatusEncodeError), err)
			}
		}
	}
}

func (replica *ReplicaInfo) acceptNewServers() {
	server, err := net.Listen("tcp", ":4000")
	if err != nil {
		replica.LOG[ERROR].Println(StatusText(StatusConnectionError), err)
		return
	}
	for {
		conn, err := server.Accept()
		if err != nil {
			replica.LOG[ERROR].Println(StatusText(StatusConnectionError), err)
			continue
		}

		conn.Close()
	}
}

func (replica *ReplicaInfo) DetermineMaster(portChannel chan int) {
	conn, err := net.Dial("tcp", ":4000")
	//if we are the master
	if err != nil {
		replica.LOG[INFO].Println("new master startup")
		replica.serverMutex.Lock()
		replica.activeServers = append(replica.activeServers, 5000)
		replica.LOG[INFO].Println(replica.activeServers)
		replica.serverMutex.Unlock()
		portChannel <- 5000
		replica.LOG[INFO].Println("test")
		//loadUsers()
		portChannel <- 0
		replica.LOG[INFO].Println("test2")
		go replica.acceptNewServers()
		go replica.sendPings()
		return
	}
	conn.Close()
	portChannel <- 0
	portChannel <- 0
}