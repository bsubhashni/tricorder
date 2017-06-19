package main

import (
	"gopkg.in/yaml.v2"
	"log"
	"io/ioutil"
	"fmt"
	"flag"
)

var coordinatorConfig Config

func loadConfig(configFile string) {
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Fatalf("Error while reading config: %v", err)
	}
	err = yaml.Unmarshal([]byte(data), &coordinatorConfig)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
}

func start() {
	fmt.Println("Starting the coordinator")
	coordinator := Coordinator{
		config:&coordinatorConfig,
		agentInfo:make(map[string]*AgentInfo),
	}
	coordinator.Run()
}

func main() {
	configFile := flag.String("config", "./config.yml", "Config file for the tricorder coordinator")
	loadConfig(*configFile)
	start()
}
