package main

import (
	pb "../../rpc"

	"net"
	"log"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"flag"
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
	lis, err := net.Listen("tcp", fmt.Sprint(":",agentConfig.Port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterAgentServiceServer(s, &Agent{
		config:&agentConfig,
	})
	// Register reflection service on gRPC server.
	reflection.Register(s)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func main() {
	configFile := flag.String("config", "./config.yml", "Config file for the tricorder agent")
	loadConfig(*configFile)
	start()
}