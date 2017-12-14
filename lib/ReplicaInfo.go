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
		IsMaster:      false,
		serverMutex:   &sync.Mutex{},
		activeServers: []int{},
        Port           -1,
		LOG:           InitLog("../../log/replica.log"),
	}
}

func (replica *ReplicaInfo) ResetServers() {
	replica.serverMutex.Lock()
	replica.activeServers = []int{}
	replica.serverMutex.Unlock()
}

func (replica *ReplicaInfo) DetermineMaster(portChannel chan int) {
	conn, err := net.Dial("tcp", ":4000")
	//if we are the master
	if err != nil {
		replica.LOG[INFO].Println("new master startup")

        replica.serverMutex.Lock()
		replica.activeServers = append(replica.activeServers, 1)
		replica.serverMutex.Unlock()

        replica.IsMaster = true
        replica.id = 1
        replica.Port = 5000

		portChannel <- 0

        go replica.acceptNewServers()
		go replica.sendPings()
		return
	}
	conn.Close()
    //replace with id
	portChannel <- 0

	portChannel <- 0
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

func (replica *ReplicaInfo) acceptNewServers(users *map[string]*UserInfo, users_lock *sync.RWMutex) {
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
        //send info
        newId = replica.generateNewId()
        

        encoder := gob.NewEncoder(conn)
        encoder.Encode(replica.activeServers)
        users_lock.RLock()
        for _, user := range users {
            err = encoder.Encode(user)
            if err != nil {
                LOG[ERROR].Println(StatusText(StatusEncodeError), err)
            }
        }
        users_lock.RUnlock()
		conn.Close()
	}
}

func (replica *ReplicaInfo) generateNewId() int {
    max := replica.activeServers[0]
    for _, elem := range replica.activeServers {
        if elem > max {
            max = elem
        }
    }
    return max
}
