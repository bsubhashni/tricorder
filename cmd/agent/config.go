package main

type AgentConfig struct {
	Port            int             `yaml:"port"`
	InterfaceConfig InterfaceConfig `yaml:"interface"`
	Mode            string          `yaml:"mode"`
}

type InterfaceConfig struct {
	Device                 string `yaml:"device"`
	CaptureType            string `yaml:"type"`
	AfPacketTragetSizeInMB int    `yaml:"targetsize"`
	Port                   int    `yaml:"port"`
}

const (
	AF_PACKET = "afpacket"
	PF_RING   = "pfring"
	PCAP      = "pcap"
)
