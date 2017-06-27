package main

import (
	pb "../../rpc"
	"flag"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"net"
	"sync"
)

var agentConfig AgentConfig

func loadConfig(configFile string) {
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Fatalf("Error while reading config: %v", err)
	}
	err = yaml.Unmarshal([]byte(data), &agentConfig)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
}

func start() {
	fmt.Printf("Starting the agent at %v", agentConfig.Port)
	lis, err := net.Listen("tcp", fmt.Sprint(":", agentConfig.Port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	agent := &Agent{
		config: &agentConfig,
		mutex: &sync.Mutex{},
	}
	agent.Initialize()
	pb.RegisterAgentServiceServer(s, agent)
	// Register reflection service on gRPC server.
	reflection.Register(s)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func main() {
	configFile := flag.String("config", "config.yml", "Config file for the tricorder agent")
	flag.Parse()
	loadConfig(fmt.Sprint("./", *configFile))
	start()
}
