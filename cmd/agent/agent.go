package main

import (
	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
	"./sniffers"
	"io"
	pb "../../rpc"
	context "golang.org/x/net/context"
	"fmt"
	"strconv"
)

type Agent struct {
	pcapHandle     		*pcap.Handle
	afpacketHandle 		*sniffers.AfpacketHandle
	config         		*AgentConfig
	isHandleAlive        	bool
	filter 			string
	dataSource 		gopacket.PacketDataSource
	streams 		map[string]*Stream
}

func (agent *Agent) handlePacket(packet gopacket.Packet) {
	transport := packet.TransportLayer()
	streamKey := transport.TransportFlow().Src().String() + transport.TransportFlow().Dst().String()
	streamKeyI := transport.TransportFlow().Dst().String() + transport.TransportFlow().Src().String()
	if agent.streams[streamKeyI] != nil {
		streamKey = streamKeyI
	}
	if agent.streams[streamKey] == nil {
		agent.streams[streamKey] = &Stream{
			currentRequests:make(map[uint32]*Command),
			currentResponses:make(map[uint32]*Command),
			src:transport.TransportFlow().Src().String(),
			dst:transport.TransportFlow().Dst().String(),
		}
	}
	agent.streams[streamKey].HandlePacket(transport.LayerPayload())
}

func (agent *Agent) startCapture() error {
	if handle, err := pcap.OpenLive("lo0", 1600, true, pcap.BlockForever); err != nil {
		panic(err)

	} else if err := handle.SetBPFFilter("tcp and port 11210"); err != nil {  // optional
		panic(err)
	} else {
		packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
		packetSource.DecodeOptions.NoCopy = true
		packetSource.DecodeOptions.Lazy = true
		agent.isHandleAlive = true
		for agent.isHandleAlive {
			packet, err := packetSource.NextPacket()
			if err == io.EOF {
				agent.isHandleAlive = false
				fmt.Printf("handle is no longer alive")
				break
			}
			agent.handlePacket(packet)
		}
	}
	return nil
}

func (agent *Agent) GetResults() ([]*pb.AgentResultsResponse_CaptureInfo) {
	var responseStats []*pb.AgentResultsResponse_CaptureInfo
	for streamkey, stream := range agent.streams {
		for _, latencyInfo := range stream.stats {
			responseStats = append(responseStats, &pb.AgentResultsResponse_CaptureInfo{
				Opaque:strconv.Itoa(int(latencyInfo.Opaque)) + streamkey,
				Latency:strconv.Itoa(latencyInfo.Latency),
			})
		}
	}
	return responseStats
}

func (agent *Agent) Hello(context.Context, *pb.CoordinatorInfo) (*pb.AgentInfo, error) {
	//TODO
	return nil, nil
}

func (agent *Agent) CaptureSignal(context.Context, *pb.CoordinatorCaptureRequest) (*pb.AgentCaptureResponse, error) {
	agent.streams = make(map[string]*Stream)
	go agent.startCapture()
	return &pb.AgentCaptureResponse{Status: "success"}, nil
}

func (agent *Agent) GoodByeSignal(context.Context, *pb.CoordinatorGoodByeRequest) (*pb.AgentGoodByeResponse, error) {
	agent.isHandleAlive = false
	agent.streams = make(map[string]*Stream)
	return &pb.AgentGoodByeResponse{Status: "success"}, nil
}


func (agent *Agent) AgentResults(context.Context, *pb.CoordinatorResultsRequest) (*pb.AgentResultsResponse, error) {
	agent.isHandleAlive = false
	responseStats := agent.GetResults()
	return &pb.AgentResultsResponse {
		Status: "success",
		Type:agent.config.Mode,
		Result: responseStats,
	}, nil
}