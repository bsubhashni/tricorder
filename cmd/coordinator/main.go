package main

import (
	"flag"
	"fmt"
	"github.com/codahale/hdrhistogram"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
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
	coordinator := Coordinator{
		config:    &coordinatorConfig,
		agentInfo: make(map[string]*AgentInfo),
		histogram: hdrhistogram.New(1, 1000*1000, 3),
	}
	coordinator.Run()
}

func main() {
	configFile := flag.String("config", "./config.yml", "Config file for the tricorder coordinator")
	flag.Parse()
	loadConfig(fmt.Sprint("./", *configFile))
	start()
}
