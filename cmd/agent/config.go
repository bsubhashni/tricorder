package main

type AgentConfig struct {
	Port		 int 		 	`yaml:"port"`
	InterfaceConfig  InterfaceConfig	`yaml:"interface"`
	Mode             string			`yaml:"mode"`
}

type InterfaceConfig struct {
	Device		 string 		`yaml:"device"`
	CaptureType 	 string 		`yaml:"capturetype,omitempty"`
}