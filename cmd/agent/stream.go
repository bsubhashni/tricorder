package main

import (
	"bytes"
)

type Stream struct {
	currentRequests  	map[uint32]*Command
	currentResponses 	map[uint32]*Command
	currentCommand		*Command
	src 			string
	dst 			string
	stats			[]LatencyInfo
}

type LatencyInfo struct {
	Opaque  		uint32
	Latency 		int
}

func (stream *Stream) collect() {
	for opaque, response := range stream.currentResponses {
		if response.isComplete() {
			if stream.currentResponses[opaque].Opcode == IGNORED {
				delete(stream.currentResponses, opaque)
			}
			if val, ok := stream.currentRequests[opaque]; !ok {
				delete(stream.currentResponses, opaque)
			} else {
				if val.Opcode == IGNORED {
					delete(stream.currentResponses, opaque)
				} else {
					latencyInfo := LatencyInfo{
						Opaque:  opaque,
						Latency: (response.CaptureTimeInNanos - val.CaptureTimeInNanos) / 1000,
					}
					stream.stats = append(stream.stats, latencyInfo)
					delete(stream.currentRequests, opaque)
					delete(stream.currentResponses, opaque)
				}
			}
		}
	}
}

func (stream *Stream) HandlePacket(data []byte) {
	if len(data) > 0 {
		if stream.currentCommand == nil {
			stream.currentCommand = NewCommand()
		}

		if err := stream.currentCommand.ReadNewPacketData(bytes.NewBuffer(data)); err != nil {
			return
		}
		if stream.currentCommand.isComplete() && stream.currentCommand.isResponse() {
			stream.currentResponses[stream.currentCommand.Opaque] = stream.currentCommand
			stream.currentCommand = nil
		} else if stream.currentCommand.isComplete() && !stream.currentCommand.isResponse() {
			stream.currentRequests[stream.currentCommand.Opaque] = stream.currentCommand
			stream.currentCommand = nil
		}
	}

	stream.collect()
}


func (stream *Stream) GetStats() []LatencyInfo {
	return stream.stats
}