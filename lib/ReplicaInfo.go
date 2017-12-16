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
    gob.Register(struct{Id int; Serverlist []int}{})


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

func (replica *ReplicaInfo) DetermineMaster(portChannel chan int, userChannel chan UserInfo, users *map[string]*UserInfo, usersLock *sync.RWMutex) {
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

	    //let backend know IsMaster is set
        portChannel <- 0
        //wait until load users is complete
        <-portChannel

        go replica.acceptNewServers(users, usersLock)
		go replica.sendPings()
		return
	}
	decoder := gob.NewDecoder(conn)
	var request CommandRequest
	err = decoder.Decode(&request)
	if err != nil {
		replica.LOG[ERROR].Println(StatusText(StatusDecodeError), err)
		panic("Can't decode info for construction")
	}
	IdAndServerList := request.Data.(struct{Id int; Serverlist []int})
	replica.id = IdAndServerList.Id
	replica.Port = IdAndServerList.Id
	replica.activeServers = IdAndServerList.Serverlist

	portChannel <- 0  // let backend know replica info has been set

	uInfo := NewUserInfo("","")
	for decoder.Decode(uInfo) == nil {
		userChannel <- *uInfo
	}
    close(userChannel)
    <-portChannel  // wait for backend to finish loading users

	conn.Close()
}

func (replica *ReplicaInfo) StartNewMaster(users *map[string]*UserInfo, usersLock *sync.RWMutex) {
	replica.LOG[INFO].Println("StartNewMasterRunning")
	go replica.sendPings()
	go replica.acceptNewServers(users, usersLock)
}

func (replica *ReplicaInfo) sendPings() {
	for {
		time.Sleep(1 * time.Second)
		replica.LOG[INFO].Println("Initiate Pinging", replica.activeServers)
		for i, serverId := range replica.activeServers {
            if serverId == replica.id {
                continue
            }
			replica.LOG[INFO].Println("Master", replica.id, "Pinging server at port", serverId)
			replica.serverMutex.Lock()
			conn, err := net.Dial("tcp", ":" + strconv.Itoa(serverId))
			if err != nil {
				conn, err = net.Dial("tcp", ":" + strconv.Itoa(serverId))
			}
			if err != nil {
				replica.LOG[WARNING].Println("Server at port", serverId, "is dead")
				replica.activeServers = append(replica.activeServers[:i], replica.activeServers[i+1:]...)
				replica.RemoveDeadServer(serverId)
				replica.serverMutex.Unlock()
				continue
			}
			replica.serverMutex.Unlock()

			command := CommandRequest{CommandSendPing, replica.id}
			encoder := gob.NewEncoder(conn)
			err = encoder.Encode(command)
			if err != nil {
				replica.LOG[ERROR].Println(StatusText(StatusEncodeError), err)
			}
			conn.Close()
		}
	}
}

func (replica *ReplicaInfo) AcceptPing(master int) {
    if master != replica.masterId {
        replica.masterId = master
        found := false
        replica.serverMutex.Lock()
        defer replica.serverMutex.Unlock()
        for _, port := range replica.activeServers {
            if port == master {
                found = true
            }
        }
        if !found {
            replica.activeServers = append(replica.activeServers, master)
        }
    }
}

func (replica *ReplicaInfo) PropagateRequest(request CommandRequest) {
    replica.LOG[INFO].Println("Propagating request", request.CommandCode)
    replica.serverMutex.Lock()
    defer replica.serverMutex.Unlock()
    for i, serverId := range replica.activeServers {
        if serverId == replica.id {
            continue
        }

        replica.LOG[INFO].Println("Sending", request.CommandCode, "to port", serverId)
        conn, err := net.Dial("tcp", ":" + strconv.Itoa(serverId))
        if err != nil {
            conn, err = net.Dial("tcp", ":" + strconv.Itoa(serverId))
        }
        if err != nil {
            replica.LOG[WARNING].Println("Server at port", serverId, "is dead")
            replica.activeServers = append(replica.activeServers[:i], replica.activeServers[i+1:]...)
            replica.RemoveDeadServer(serverId)
            continue
        }
        encoder := gob.NewEncoder(conn)
        err = encoder.Encode(request)
        if err != nil {
            replica.LOG[ERROR].Println(StatusText(StatusEncodeError), err)
        }

        var response CommandResponse
        decoder := gob.NewDecoder(conn)
        err = decoder.Decode(&response)
        if err != nil {
            replica.LOG[ERROR].Println(StatusText(StatusDecodeError), err)
        }
        if !response.Success {
            replica.LOG[ERROR].Println("Replica at port:", serverId, "failed to run command:", request.CommandCode)
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
        replica.activeServers = append(replica.activeServers, newId)

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
            err = encoder.Encode(*user)
            if err != nil {
                replica.LOG[ERROR].Println(StatusText(StatusEncodeError), err)
            }
        }
        usersLock.RUnlock()
		conn.Close()
        replica.sendNewServer(newId)
	}
}

func (replica *ReplicaInfo) sendNewServer(newId int) {
    replica.LOG[INFO].Println("Send new server", newId)
    for i, serverId := range replica.activeServers {
        if serverId == replica.id || serverId == newId{
            continue
        }
		replica.LOG[INFO].Println("replica id:", replica.id, "port", serverId)
		replica.serverMutex.Lock()
		conn, err := net.Dial("tcp", ":" + strconv.Itoa(serverId))
		if err != nil {
			conn, err = net.Dial("tcp", ":" + strconv.Itoa(serverId))
		}
		if err != nil {
			replica.LOG[WARNING].Println("Server at port", serverId, "is dead")
			replica.activeServers = append(replica.activeServers[:i], replica.activeServers[i+1:]...)
			replica.RemoveDeadServer(serverId)
			replica.serverMutex.Unlock()
			continue
		}
		replica.serverMutex.Unlock()

		command := CommandRequest{CommandNewServer, newId}
		encoder := gob.NewEncoder(conn)
		err = encoder.Encode(command)
		if err != nil {
			replica.LOG[ERROR].Println(StatusText(StatusEncodeError), err)
	    }
	    conn.Close()
    }
}

func (replica *ReplicaInfo) RemoveDeadServer(deadId int) {
	replica.LOG[INFO].Println("Send dead server", deadId)
	for _, port := range replica.activeServers {
		if port == replica.id || port == deadId{
			continue
		}
		replica.LOG[INFO].Println("replica id:", replica.id, "port", port)
		replica.serverMutex.Lock()
		conn, err := net.Dial("tcp", ":" + strconv.Itoa(port))
		if err != nil {
			conn, err = net.Dial("tcp", ":" + strconv.Itoa(port))
		}
		if err != nil {  // server may be dead avoid recursing and detect again later
			replica.serverMutex.Unlock()
			continue
		}
		replica.serverMutex.Unlock()

		command := CommandRequest{CommandDeadServer, deadId}
		encoder := gob.NewEncoder(conn)
		err = encoder.Encode(command)
		if err != nil {
			replica.LOG[ERROR].Println(StatusText(StatusEncodeError), err)
		}
		conn.Close()
	}
}

func (replica *ReplicaInfo) OnNewServer(newId int) {
	replica.LOG[INFO].Println("New Server:", newId)
    replica.serverMutex.Lock()
    replica.activeServers = append(replica.activeServers, newId)
    replica.serverMutex.Unlock()
}

func (replica *ReplicaInfo) OnDeadServer(deadId int) {
	replica.LOG[WARNING].Println("Dead Server:", deadId)
	replica.serverMutex.Lock()
	for i, elem := range replica.activeServers {
		if elem == deadId {
			replica.activeServers = append(replica.activeServers[:i], replica.activeServers[i+1:]...)
		}
	}
	replica.serverMutex.Unlock()
}

func (replica *ReplicaInfo) HoldElection(masterChan chan int) {
	i := 0
	replica.serverMutex.Lock()
	for replica.activeServers[i] != replica.masterId {
		i++
	}
	replica.activeServers = append(replica.activeServers[:i],replica.activeServers[i+1:]...)
	min := replica.activeServers[0]
	for _, elem := range replica.activeServers {
		if elem < min {
			min = elem
		}
	}
    replica.masterId = min
	replica.serverMutex.Unlock()
	if min == replica.id {
		replica.LOG[INFO].Println("This replica is taking over as master")
		replica.IsMaster = true
		replica.Port = 5000
	}
	masterChan <- 0
}

func (replica *ReplicaInfo) generateNewId() int {
    max := replica.activeServers[0]
    for _, elem := range replica.activeServers {
        if elem > max {
            max = elem
        }
    }
    return max + 1
}
