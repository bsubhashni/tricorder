package main

import (
	"bytes"
	"sync"
)

type Stream struct {
	mutex				*sync.Mutex
	currentRequests  	map[uint32]*Command
	currentResponses 	map[uint32]*Command
	currentCommand   	*Command
	src              	string
	dst              	string
	latencyInfo         []LatencyInfo
}

type LatencyInfo struct {
	Opaque  uint32
	Latency int64
	Key 	string
}

func (stream *Stream) collect() {
	for opaque, response := range stream.currentResponses {
		if response.isComplete() {
			if stream.currentResponses[opaque].opcode == IGNORED {
				delete(stream.currentResponses, opaque)
			}
			if request, ok := stream.currentRequests[opaque]; !ok {
				delete(stream.currentResponses, opaque)
			} else {
				if request.opcode == IGNORED {
					delete(stream.currentResponses, opaque)
				} else {
					latencyInfo := LatencyInfo{
						Opaque:  opaque,
						Latency: (response.captureTimeInNanos - request.captureTimeInNanos) / 1000,
						Key: string(request.key),
					}
					stream.latencyInfo = append(stream.latencyInfo, latencyInfo)
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
			stream.currentResponses[stream.currentCommand.opaque] = stream.currentCommand
			stream.currentCommand = nil
		} else if stream.currentCommand.isComplete() && !stream.currentCommand.isResponse() {
			stream.currentRequests[stream.currentCommand.opaque] = stream.currentCommand
			stream.currentCommand = nil
		}
	}

	stream.collect()
}