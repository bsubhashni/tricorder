package main

import (
	"google.golang.org/grpc"
	"fmt"
	"sync"
	pb "../../rpc"
	"log"
	"context"
	"time"
	"os"
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
)


type AgentsConfig struct {
	agent		map[string]string
}

type Coordinator struct {
	config  	*Config
	agentInfo 	map[string]*AgentInfo
	db              *sql.DB
}


type AgentInfo struct {
	index 		int
	hostname 	string
	connected 	Status
	conn 		*grpc.ClientConn
	client 		pb.AgentServiceClient
	results		[]LatencyInfo
}

type Status int

const (
	CONNECTED Status = iota
	DISCONNECTED
)

type LatencyInfo struct {
	nodeType string
	key string
	latency string
}


func (coordinator *Coordinator) setupStore() {
	file := coordinator.config.History.FileName
	os.Remove(file)
	db, err := sql.Open("sqlite3", file)
	if err != nil {
		log.Fatal(err)
	}
	sqlStmt := `
	create table ResultStats (id text not null primary key, type text, latency text);
	delete from ResultStats;
	`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Fatalf("%q: %s\n", err, sqlStmt)
		return
	}
	coordinator.db = db

}

func connectToAgent(hostName string) (*grpc.ClientConn, error) {
	return grpc.Dial(hostName, grpc.WithInsecure())
}

func (coordinator *Coordinator) ConnectToAgents() {
	for ii, hostName := range coordinator.config.Agents {

		conn, err := connectToAgent(hostName)
		if err != nil {
			log.Fatalf("Unable to connect to the Agent %v %v", err, hostName)
		}
		fmt.Println("Connected to the agent", hostName)
		coordinator.agentInfo[hostName] = &AgentInfo{
			index: ii,
			hostname:hostName,
			connected: CONNECTED,
			conn: conn,
			client: pb.NewAgentServiceClient(conn),
		}
	}
}

func (coordinator *Coordinator) startCapture(wg *sync.WaitGroup, agentInfo *AgentInfo)  {
	_, err := agentInfo.client.CaptureSignal(context.Background(), &pb.CoordinatorCaptureRequest{SequencePrefix:int32(agentInfo.index)})
	if err != nil {
		log.Fatalf("Unable to start capture on agent %s due to %v", agentInfo.hostname, err)
	}
	wg.Done()
}

func (coordinator *Coordinator) StartCapture() {
	wg := sync.WaitGroup{}
	wg.Add(len(coordinator.agentInfo))
	for _, agent := range coordinator.agentInfo {
		go coordinator.startCapture(&wg, agent)
	}
	wg.Wait()
}

func (coordinator *Coordinator) mergeAndStore() {
	tx, err := coordinator.db.Begin()
	if err != nil {
		log.Fatal(err)
	}

	stmt, err := tx.Prepare("insert into ResultStats(id, type, latency) values(?, ?, ?)")
	if err != nil {
		log.Fatal(err)
	}

	for _, agentInfo := range coordinator.agentInfo {
		fmt.Println("Got", len(agentInfo.results), "results from", agentInfo.hostname)

		for _, row := range agentInfo.results {
			_, err = stmt.Exec(row.key, row.nodeType, row.latency)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
	tx.Commit()
}

func (coordinator *Coordinator) getResults(wg *sync.WaitGroup, agentInfo *AgentInfo) {
	if response, err := agentInfo.client.AgentResults(context.Background(), &pb.CoordinatorResultsRequest{}); err != nil {
		log.Fatalf("Unable to get results from agent %s due to %v", agentInfo.hostname, err)
	} else {
		for _, row := range response.Result {
			agentInfo.results = append(agentInfo.results, LatencyInfo {
				nodeType:response.Type,
				key:row.Opaque,
				latency:row.Latency,
			})
		}
	}
	wg.Done()
}

func (coordinator *Coordinator) GetResults() {
	wg := sync.WaitGroup{}
	wg.Add(len(coordinator.agentInfo))
	for _, agent := range coordinator.agentInfo {
		go coordinator.getResults(&wg, agent)
	}
	wg.Wait()
}

func (coordinator *Coordinator) sayGoodBye(wg *sync.WaitGroup, agentInfo *AgentInfo) {
	_, err := agentInfo.client.AgentResults(context.Background(), &pb.CoordinatorResultsRequest{})
	if err != nil {
		log.Fatalf("Unable to get results from agent %s due to %v", agentInfo.hostname, err)
	}
	wg.Done()
}


func (coordinator *Coordinator) sayGoodbye(wg *sync.WaitGroup, agentInfo *AgentInfo) {
	_, err := agentInfo.client.GoodByeSignal(nil, &pb.CoordinatorGoodByeRequest{})
	if err != nil {
		log.Fatalf("Unable to say good bye to agent %s due to %v", agentInfo.hostname, err)
	}
	wg.Done()
}

func (coordinator *Coordinator) ShutDown() {
	wg := sync.WaitGroup{}
	wg.Add(len(coordinator.agentInfo))
	for _, agent := range coordinator.agentInfo {
		go coordinator.sayGoodbye(&wg, agent)
	}
	wg.Wait()
}

func (coordinator *Coordinator) Run() {
	coordinator.setupStore()
	currentTime := time.Now().Nanosecond() / (1000 * 1000)
	maxRunTime := currentTime + coordinator.config.History.Period * 60 * 1000
	coordinator.ConnectToAgents()

	for currentTime < maxRunTime {

		coordinator.StartCapture()
		time.Sleep(time.Duration(coordinator.config.Capture.Period) * time.Millisecond)
		coordinator.GetResults()
		//storeResults in separate goroutine
		go coordinator.mergeAndStore()
		time.Sleep(time.Duration(coordinator.config.Capture.Interval) * time.Millisecond)
		currentTime = time.Now().Nanosecond() / (1000 * 1000)
	}

	coordinator.ShutDown()
}