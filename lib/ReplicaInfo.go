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
	IsMaster      bool
	serverMutex   *sync.Mutex
	activeServers []int
	Port		  int
	LOG           map[int]*log.Logger
}

func NewReplica() ReplicaInfo {
    gob.Register([]Post{})
    gob.Register(struct{Username, Password string}{})
    gob.Register(struct{Username1, Username2 string}{})
    gob.Register(struct{Searcher, Target string}{})
    gob.Register(struct{Username, Post string}{})


	return ReplicaInfo{
		id:            -1,
		masterId:      -1,
		IsMaster:      false,
		serverMutex:   &sync.Mutex{},
		activeServers: []int{},
        Port:           -1,
		LOG:           InitLog("../../log/replica.log"),
	}
}

func (replica *ReplicaInfo) ResetServers() {
	replica.serverMutex.Lock()
	replica.activeServers = []int{}
	replica.serverMutex.Unlock()
}

func (replica *ReplicaInfo) DetermineMaster(portChannel chan int, userChannel chan *UserInfo, users *map[string]*UserInfo, usersLock *sync.RWMutex) {
	conn, err := net.Dial("tcp", ":4000")
	//if we are the master
	if err != nil {
		replica.LOG[INFO].Println("new master startup")

        replica.serverMutex.Lock()
		replica.activeServers = append(replica.activeServers, 5001)
		replica.serverMutex.Unlock()

        replica.IsMaster = true
        replica.id = 5001
        replica.Port = 5000

		portChannel <- 0

        go replica.acceptNewServers(users, usersLock)
		go replica.sendPings()
		return
	}
	portChannel <- 0
	decoder := gob.NewDecoder(conn)
	var request CommandRequest
	err = decoder.Decode(&request)
	if err != nil {
		replica.LOG[ERROR].Println(StatusText(StatusDecodeError), err)
		panic("Can't decode info for construction")
	}

	uInfo := NewUserInfo("","")
	for decoder.Decode(uInfo) != nil {
		userChannel <- uInfo
	}
	close(userChannel)
	conn.Close()
}

func (replica *ReplicaInfo) sendPings() {
	for {
		time.Sleep(3 * time.Second)
		replica.LOG[INFO].Println("Initiate Pinging")
		for i, port := range replica.activeServers {
            if port == replica.id {
                continue
            }
			replica.LOG[INFO].Println("Pinging server at port", port)
			replica.serverMutex.Lock()
			conn, err := net.Dial("tcp", ":" + strconv.Itoa(port))
			if err != nil {
				conn, err = net.Dial("tcp", ":" + strconv.Itoa(port))
			}
			if err != nil {
				replica.LOG[WARNING].Println("Server at port", port, "is dead")
				replica.activeServers = append(replica.activeServers[:i], replica.activeServers[i+1:]...)
				replica.serverMutex.Unlock()
				continue
			}
			replica.serverMutex.Unlock()

			command := CommandRequest{CommandSendPing, nil}
			encoder := gob.NewEncoder(conn)
			err = encoder.Encode(command)
			if err != nil {
				replica.LOG[ERROR].Println(StatusText(StatusEncodeError), err)
			}
			conn.Close()
		}
	}
}

func (replica *ReplicaInfo) PropagateRequest(request CommandRequest) {
    replica.LOG[INFO].Println("Propagating request", request.CommandCode)
    replica.serverMutex.Lock()
    defer replica.serverMutex.Unlock()
    for i, port := range replica.activeServers {
        if port == replica.Port {
            continue
        }

        replica.LOG[INFO].Println("Sending", request.CommandCode, "to port", port)
        conn, err := net.Dial("tcp", ":" + strconv.Itoa(port))
        if err != nil {
            conn, err = net.Dial("tcp", ":" + strconv.Itoa(port))
        }
        if err != nil {
            replica.LOG[WARNING].Println("Server at port", port, "is dead")
            replica.activeServers = append(replica.activeServers[:i], replica.activeServers[i+1:]...)
            continue
        }
        encoder := gob.NewEncoder(conn)
        err = encoder.Encode(request)
        if err != nil {
            replica.LOG[ERROR].Println(StatusText(StatusEncodeError), err)
        }
        conn.Close()
    }
}

func (replica *ReplicaInfo) acceptNewServers(users *map[string]*UserInfo, usersLock *sync.RWMutex) {
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
        newId := replica.generateNewId()

        request := CommandRequest{CommandConstructFilesystem, struct {
			Id         int
			Serverlist []int
		}{
			newId,
			replica.activeServers,
		}}

        encoder := gob.NewEncoder(conn)
        encoder.Encode(request)
        usersLock.RLock()
        for _, user := range *users {
            err = encoder.Encode(user)
            if err != nil {
                replica.LOG[ERROR].Println(StatusText(StatusEncodeError), err)
            }
        }
        usersLock.RUnlock()
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
