// +build !linux

package sniffers

import (
	"fmt"
	"github.com/google/gopacket"
	"time"
)

type AfpacketHandle struct {
}

func NewAfpacketHandle(device string, snaplen int, blockSize int, numBlocks int,
	timeout time.Duration) (*AfpacketHandle, error) {
	return nil, fmt.Errorf("Afpacket sniffing is only available on Linux")
}

func (h *AfpacketHandle) SetBPFFilter(expr string) (_ error) {
	return fmt.Errorf("Afpacket  sniffing is only available on Linux")
}

func (h *AfpacketHandle) GetPacketSource() *gopacket.PacketSource {
	return nil
}

func (h *AfpacketHandle) Close() {
}
