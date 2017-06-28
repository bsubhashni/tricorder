/*
 * Copyright (c) 2017 Couchbase, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package main

import (
	pb "../../rpc"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/codahale/hdrhistogram"
	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
	"google.golang.org/grpc"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

type AgentsConfig struct {
	agent map[string]string
}

type Coordinator struct {
	config             *Config
	agentsInfo         map[string]*AgentInfo
	db                 *sql.DB
	insertStatementStr string
	histogram          *hdrhistogram.Histogram
}

type AgentInfo struct {
	index    int
	hostname string
	conn     *grpc.ClientConn
	client   pb.AgentServiceClient
	results  map[string]*pb.AgentResultsResponse_CaptureInfo
}

type LatencyInfo struct {
	nodeType string
	opaque   string
	latency  string
	key      string
}

func (coordinator *Coordinator) homeHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	if html, err := ioutil.ReadFile("./graphplotter/index.html"); err != nil {
		log.Fatal(err)
	} else {
		buffer := bytes.NewBuffer(make([]byte, 0, 1024))
		jsonStr, err := coordinator.getFullCaptureFromDb()
		if err != nil {
			log.Fatal(err)
		}

		buffer.WriteString("<script type=\"text/javascript\">")
		buffer.WriteString("var data=")
		buffer.WriteString(jsonStr)
		buffer.WriteString(";")
		buffer.WriteString("var yMax=")
		buffer.WriteString(strconv.FormatInt(coordinator.getMaxLatency(), 10))
		buffer.WriteString(";")
		buffer.WriteString("</script>")
		buffer.Write(html)

		w.Write(buffer.Bytes())
	}
}

func (coordinator *Coordinator) startRestServer() {
	r := mux.NewRouter()
	r.HandleFunc("/", coordinator.homeHandler)
	http.Handle("/", r)

	srv := &http.Server{
		Handler: r,
		Addr:    fmt.Sprint(":", coordinator.config.RestPort),
	}
	log.Fatal(srv.ListenAndServe())
}

func (coordinator *Coordinator) setupStore() {
	file := coordinator.config.History.FileName
	os.Remove(file)
	db, err := sql.Open("sqlite3", file)
	if err != nil {
		log.Fatal(err)
	}
	buffer := bytes.NewBuffer(make([]byte, 0, 1024))
	for _, agent := range coordinator.agentsInfo {
		buffer.WriteString(fmt.Sprint("agent", agent.index))
		buffer.WriteString(" text")
		if agent.index < (len(coordinator.agentsInfo) - 1) {
			buffer.WriteString(", ")
		}
	}
	s := buffer.String()
	sqlStmt := fmt.Sprintf("create table CaptureResults (opaque_streamId text not null, timestamp integer, %v); delete from CaptureResults;", s)
	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Fatalf("%q: %s\n", err, sqlStmt)
		return
	}
	fieldsBuffer := bytes.NewBuffer(make([]byte, 0, 1024))
	argsBuffer := bytes.NewBuffer(make([]byte, 0, 1024))
	for _, agent := range coordinator.agentsInfo {
		fieldsBuffer.WriteString(fmt.Sprint("agent", agent.index))
		argsBuffer.WriteString("?")
		if agent.index < (len(coordinator.agentsInfo) - 1) {
			fieldsBuffer.WriteString(", ")
			argsBuffer.WriteString(", ")
		}
	}
	f := fieldsBuffer.String()
	a := argsBuffer.String()

	statementStr := fmt.Sprintf("insert into CaptureResults(opaque_streamId, timestamp, %v) values(?, ?, %v)", f, a)
	coordinator.insertStatementStr = statementStr
	coordinator.db = db
}

func (coordinator *Coordinator) storeFlusher() {
	currentTime := time.Now().UnixNano() / int64(time.Millisecond)
	maxHistoryTime := currentTime + int64(coordinator.config.History.Period*60)
	for {
		if currentTime < maxHistoryTime {
			time.Sleep(time.Second * time.Duration(maxHistoryTime-currentTime))
		}
		sqlStmt := `delete from CaptureResults;`
		_, err := coordinator.db.Exec(sqlStmt)
		if err != nil {
			log.Fatalf("%q: %s\n", err, sqlStmt)
			return
		}

	}
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
		coordinator.agentsInfo[hostName] = &AgentInfo{
			index:    ii,
			hostname: hostName,
			conn:     conn,
			client:   pb.NewAgentServiceClient(conn),
		}
	}
}

func (coordinator *Coordinator) startCapture(wg *sync.WaitGroup, agentInfo *AgentInfo) {
	_, err := agentInfo.client.CaptureSignal(context.Background(), &pb.CoordinatorCaptureRequest{SequencePrefix: int32(agentInfo.index)})
	if err != nil {
		log.Fatalf("Unable to start capture on agent %s due to %v", agentInfo.hostname, err)
	}
	wg.Done()
}

func (coordinator *Coordinator) StartCapture() {
	wg := sync.WaitGroup{}
	wg.Add(len(coordinator.agentsInfo))
	for _, agent := range coordinator.agentsInfo {
		go coordinator.startCapture(&wg, agent)
	}
	wg.Wait()
}

func (coordinator *Coordinator) mergeAndStore() {
	tx, err := coordinator.db.Begin()
	if err != nil {
		log.Fatal(err)
	}

	stmt, err := tx.Prepare(coordinator.insertStatementStr)

	if err != nil {
		log.Fatalf("%v", err)
	}

	var agentsInfo []*AgentInfo
	for _, agentInfo := range coordinator.agentsInfo {
		agentsInfo = append(agentsInfo, agentInfo)
	}
	timestamp := time.Now().Unix() * 1000

	for rowKey, row := range agentsInfo[0].results {
		lat, _ := strconv.ParseInt(row.Oplatency, 10, 64)
		coordinator.histogram.RecordValue(lat)
		var args []interface{}
		args = append(args, rowKey)
		args = append(args, timestamp)
		args = append(args, row.Oplatency)
		found := false
		for i := 1; i < len(agentsInfo); i++ {
			agent := agentsInfo[i]
			if row := agent.results[rowKey]; row != nil {
				args = append(args, row.Oplatency)
				lat, _ := strconv.ParseInt(row.Oplatency, 10, 64)
				coordinator.histogram.RecordValue(lat)
				found = true
			}
		}
		if !found {
			break
		}

		_, err = stmt.Exec(args...)
		if err != nil {
			log.Fatal(err)
		}
	}

	for _, agentInfo := range agentsInfo {
		agentInfo.results = nil
	}

	tx.Commit()
}

func (coordinator *Coordinator) getFullCaptureFromDb() (string, error) {
	tx, err := coordinator.db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	rows, err := tx.Query("select * from CaptureResults;")
	defer rows.Close()
	if err != nil {
		return "", err
	}
	columns, err := rows.Columns()
	if err != nil {
		return "", err
	}
	count := len(columns)
	tableData := make([]map[string]interface{}, 0)
	values := make([]interface{}, count)
	valuePtrs := make([]interface{}, count)
	for rows.Next() {
		for i := 0; i < count; i++ {
			valuePtrs[i] = &values[i]
		}
		rows.Scan(valuePtrs...)
		entry := make(map[string]interface{})
		for i, col := range columns {
			var v interface{}
			val := values[i]
			b, ok := val.([]byte)
			if ok {
				v = string(b)
			} else {
				v = val
			}
			entry[col] = v
		}
		tableData = append(tableData, entry)
	}
	tx.Commit()

	jsonData, err := json.Marshal(tableData)
	if err != nil {
		return "", err
	}
	return string(jsonData), nil
}

func (coordinator *Coordinator) getMaxLatency() int64 {
	return coordinator.histogram.Max()
}

func (coordinator *Coordinator) getResults(wg *sync.WaitGroup, agentInfo *AgentInfo) {
	if response, err := agentInfo.client.AgentResults(context.Background(), &pb.CoordinatorResultsRequest{}); err != nil {
		log.Fatalf("Unable to get results from agent %s due to %v", agentInfo.hostname, err)
	} else {
		fmt.Println(time.Now().UTC().Format(time.RFC3339Nano)+" Got", len(response.CaptureMap), " results from ", agentInfo.hostname)
		agentInfo.results = response.CaptureMap
	}
	wg.Done()
}

func (coordinator *Coordinator) GetResults() {
	wg := sync.WaitGroup{}
	wg.Add(len(coordinator.agentsInfo))
	for _, agent := range coordinator.agentsInfo {
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
	wg.Add(len(coordinator.agentsInfo))
	for _, agent := range coordinator.agentsInfo {
		go coordinator.sayGoodbye(&wg, agent)
	}
	wg.Wait()
}

func (coordinator *Coordinator) Run() {
	coordinator.ConnectToAgents()
	coordinator.setupStore()
	go coordinator.storeFlusher()

	for {
		coordinator.StartCapture()
		time.Sleep(time.Duration(coordinator.config.Capture.Period) * time.Millisecond)
		coordinator.GetResults()
		go coordinator.mergeAndStore()
		time.Sleep(time.Duration(coordinator.config.Capture.Interval) * time.Millisecond)
	}
	coordinator.ShutDown()
}