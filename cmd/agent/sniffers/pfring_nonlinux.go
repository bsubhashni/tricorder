// +build !linux

package sniffers

import (
	"fmt"
	"github.com/google/gopacket"
)

type PfringHandle struct {
}

func NewPfringHandle(device string, snaplen int, promisc bool) (*PfringHandle, error) {
	return nil, fmt.Errorf("PF_RING sniffing is only available on Linux")
}

func (h *PfringHandle) SetBPFFilter(expr string) (_ error) {
	return fmt.Errorf("PF_RING sniffing is only available on Linux")
}

func (h *PfringHandle) Enable() (_ error) {
	return fmt.Errorf("PF_RING sniffing is only available on Linux")
}

func (h *PfringHandle) GetPacketSource() *gopacket.PacketSource {
	return nil
}

func (h *PfringHandle) Close() {
}
