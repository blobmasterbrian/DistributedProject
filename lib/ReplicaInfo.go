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

//creates a new Replica Info object with dummy values for id
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

//clears the list of active servers
func (replica *ReplicaInfo) ResetServers() {
	replica.serverMutex.Lock()
	replica.activeServers = []int{}
	replica.serverMutex.Unlock()
}

//Checks to see if there is a current running master server.  If there is not, set yourself as the master server
//If there is a master, query the master for user information and active server information
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

//spawns off goroutines for sending pings to other servers and to accept new servers
func (replica *ReplicaInfo) StartNewMaster(users *map[string]*UserInfo, usersLock *sync.RWMutex) {
	replica.LOG[INFO].Println("StartNewMasterRunning")
	go replica.sendPings()
	go replica.acceptNewServers(users, usersLock)
}

//send pings to active servers to show that the master is still running
func (replica *ReplicaInfo) sendPings() {
	for {
		time.Sleep(1 * time.Second)
		replica.LOG[INFO].Println("Initiate Pinging", replica.activeServers)
		var deadServers []int
		replica.serverMutex.Lock()
		for _, serverId := range replica.activeServers {
            if serverId == replica.id {
                continue
            }
			replica.LOG[INFO].Println("Master", replica.id, "Pinging server at port", serverId)
			conn, err := net.Dial("tcp", ":" + strconv.Itoa(serverId))
			if err != nil {
				conn, err = net.Dial("tcp", ":" + strconv.Itoa(serverId))
			}
			if err != nil {
				replica.LOG[WARNING].Println("Server at port", serverId, "is dead")
				deadServers = append(deadServers, serverId)
				continue
			}

			command := CommandRequest{CommandSendPing, replica.id}
			encoder := gob.NewEncoder(conn)
			err = encoder.Encode(command)
			if err != nil {
				replica.LOG[ERROR].Println(StatusText(StatusEncodeError), err)
			}
			conn.Close()
		}
		replica.serverMutex.Unlock()
		for _, dead := range deadServers {
			replica.RemoveDeadServer(dead)
		}
	}
}

//accept a ping from the master and set the master if it is different from the previously stored master
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

//send a command request from the master to the replica servers, the master behaves as the frontend would to the
//master server, this ensures data consistancy between the master and the replicas
func (replica *ReplicaInfo) PropagateRequest(request CommandRequest) {
    replica.LOG[INFO].Println("Propagating request", request.CommandCode)
    var deadServers []int
    replica.serverMutex.Lock()
    for _, serverId := range replica.activeServers {
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
            deadServers = append(deadServers, serverId)
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
    replica.serverMutex.Unlock()
    for _, dead := range deadServers {
    	replica.RemoveDeadServer(dead)
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
        replica.serverMutex.Lock()
        replica.activeServers = append(replica.activeServers, newId)

        request := CommandRequest{CommandConstructFilesystem, struct {
			Id         int
			Serverlist []int
		}{
			newId,
			replica.activeServers,
		}}
		replica.serverMutex.Unlock()

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
    var deadServers []int
    replica.serverMutex.Lock()
    for _, serverId := range replica.activeServers {
        if serverId == replica.id || serverId == newId {
            continue
        }
		replica.LOG[INFO].Println("replica id:", replica.id, "port", serverId)
		conn, err := net.Dial("tcp", ":" + strconv.Itoa(serverId))
		if err != nil {
			conn, err = net.Dial("tcp", ":" + strconv.Itoa(serverId))
		}
		if err != nil {
			replica.LOG[WARNING].Println("Server at port", serverId, "is dead")
			deadServers = append(deadServers, serverId)
			continue
		}

		command := CommandRequest{CommandNewServer, newId}
		encoder := gob.NewEncoder(conn)
		err = encoder.Encode(command)
		if err != nil {
			replica.LOG[ERROR].Println(StatusText(StatusEncodeError), err)
	    }
	    conn.Close()
    }
    replica.serverMutex.Unlock()
    for _, dead := range deadServers {
    	replica.RemoveDeadServer(dead)
	}
}

func (replica *ReplicaInfo) RemoveDeadServer(deadId int) {
	replica.LOG[INFO].Println("Send dead server", deadId)

	replica.serverMutex.Lock()
	defer replica.serverMutex.Unlock()
	for i, serverId := range replica.activeServers {
		if serverId == deadId {
			replica.activeServers = append(replica.activeServers[:i], replica.activeServers[i+1:]...)
		}
	}

	for _, serverId := range replica.activeServers {
		if serverId == replica.id || serverId == deadId {
			continue
		}
		replica.LOG[INFO].Println("replica id:", replica.id, "port", serverId)
		conn, err := net.Dial("tcp", ":" + strconv.Itoa(serverId))
		if err != nil {
			conn, err = net.Dial("tcp", ":" + strconv.Itoa(serverId))
		}
		if err != nil {  // server may be dead avoid recursing and detect again later
			continue
		}

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
    replica.LOG[INFO].Println("Updated Server List:", replica.activeServers)
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
	replica.LOG[INFO].Println("Updated Server List:", replica.activeServers)
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
