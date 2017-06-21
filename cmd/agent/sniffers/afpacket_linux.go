// +build linux

package sniffers

import (
	"github.com/google/gopacket"
	"github.com/google/gopacket/afpacket"
	"time"
)

type AfpacketHandle struct {
	TPacket *afpacket.TPacket
}

func NewAfpacketHandle(device string, snaplen int, block_size int, num_blocks int,
	timeout time.Duration) (*AfpacketHandle, error) {

	h := &AfpacketHandle{}
	var err error

	if device == "any" {
		h.TPacket, err = afpacket.NewTPacket(
			afpacket.OptFrameSize(snaplen),
			afpacket.OptBlockSize(block_size),
			afpacket.OptNumBlocks(num_blocks),
			afpacket.OptPollTimeout(timeout))
	} else {
		h.TPacket, err = afpacket.NewTPacket(
			afpacket.OptInterface(device),
			afpacket.OptFrameSize(snaplen),
			afpacket.OptBlockSize(block_size),
			afpacket.OptNumBlocks(num_blocks),
			afpacket.OptPollTimeout(timeout))
	}

	return h, err
}

func (h *AfpacketHandle) SetBPFFilter(expr string) (_ error) {
	return h.TPacket.SetBPFFilter(expr)
}

func (h *AfpacketHandle) Close() {
	h.TPacket.Close()
}

func (h *AfpacketHandle) GetPacketSource() *gopacket.PacketSource {
	return gopacket.NewPacketSource(TPacket, layers.LinkTypeEthernet)
}
