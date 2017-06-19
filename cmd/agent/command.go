package main


import (
	"time"
	"bytes"
	"log"
	"encoding/binary"
	"io"
)

type Command struct {
	state 			ParserState
	commandType 		CommandType
	Opcode 			string
	magic			uint8
	Opaque			uint32
	keyLength		uint16
	extrasLength		uint8
	valueLength		uint32
	cas			uint32
	partial			[]byte
	CaptureTimeInNanos	int
}

type ParserState int

const (
	parseStateHeader ParserState = iota
	parseStateExtras
	parseStateKey
	parseStateValue
	parseStateComplete
)

type CommandType int

const (
	REQUEST CommandType = iota
	RESPONSE
)

type Opcode string

const (
	GET = "0"
	SET = "1"
	IGNORED = "IGNORED"

)

func NewCommand() *Command {
	return &Command{
		state:       		parseStateHeader,
		CaptureTimeInNanos: 	time.Now().Nanosecond(),
	}
}

func (c *Command) ReadNewPacketData(data *bytes.Buffer) error {
	if c.state == parseStateHeader {
		partialLen := len(c.partial)
		if data.Len() < 24 && partialLen == 0 {
			data.Read(c.partial)
			return io.EOF
		} else if data.Len() < 24 && partialLen > 0 {
			if b, err := data.ReadByte(); err != nil {
				if err == io.EOF {
					return err
				} else {
					log.Fatalf("Failed parsing packet at partial %v due to %v", c.state, err)
				}
			} else {
				c.partial = append(c.partial, b)
			}
			return io.EOF
		}

		var header bytes.Buffer

		if len(c.partial) > 0 {
			if n, err := header.Write(c.partial); err != nil {
				log.Fatalf("Failed parsing packet at partial %v due to %v", c.state, err)
			} else {
				header.Write(data.Next(24-n))
			}
		} else {
			header.Write(data.Next(24))
		}

		if magic, err := header.ReadByte(); err != nil {
			log.Fatal("Failed parsing packet at magic %v due to %v", c.state, err)
		} else {
			c.magic = magic
			if c.magic == 0x80 {
				c.commandType = REQUEST
			} else if c.magic == 0x81 {
				c.commandType = RESPONSE
			}
		}


		if opcode, err := header.ReadByte(); err != nil {
			log.Fatal("Failed parsing packet opcode %v due to %v", c.state, err)
		} else {
			if opcode == 0x0 {
				c.Opcode = GET
			} else if opcode == 0x1 {
				c.Opcode = SET
			} else {
				c.Opcode = IGNORED
			}
		}

		keyLenBytes := header.Next(2)
		c.keyLength = binary.BigEndian.Uint16(keyLenBytes)

		extrasLenBytes, _ := header.ReadByte()
		c.extrasLength = extrasLenBytes

		header.Next(1) //datatype
		header.Next(2) //vbucket or status

		totalBodyLength := binary.BigEndian.Uint32(header.Next(4))
		c.valueLength = totalBodyLength - uint32(c.keyLength) - uint32(c.extrasLength)

		opaqueBytes := header.Next(4)
		c.Opaque = binary.BigEndian.Uint32(opaqueBytes)
		header.Next(2) //cas

		if data.Len() > 0 && c.extrasLength > 0 {
			c.state = parseStateExtras
		} else if data.Len() > 0 && c.extrasLength == 0 && c.keyLength > 0 {
			c.state = parseStateKey
		} else if data.Len() > 0 && c.extrasLength == 0 && c.valueLength > 0 {
			c.state = parseStateValue
		} else {
			c.state = parseStateComplete
		}
		c.partial = nil

	}

	if c.state == parseStateExtras {
		extrasLen := int(c.extrasLength)

		if data.Len() >= extrasLen {
			data.Next(extrasLen)
			if c.keyLength > 0 {
				c.state = parseStateKey
			} else if c.valueLength > 0 {
				c.state = parseStateValue
			} else {
				c.state = parseStateComplete
			}
		} else {
			available := data.Len()
			data.Next(available)
			c.extrasLength -= uint8(available)
			return io.EOF
		}
	}

	if c.state == parseStateKey {
		keyLen := int(c.keyLength)

		if data.Len() >= keyLen {
			data.Next(keyLen)
			if c.valueLength > 0 {
				c.state = parseStateValue
			} else {
				c.state = parseStateComplete
			}
		} else {
			available := data.Len()
			data.Next(available)
			c.keyLength -= uint16(available)
			return io.EOF
		}

	}

	if c.state == parseStateValue {
		valueLen := int(c.valueLength)

		if data.Len() >= valueLen {
			data.Next(valueLen)
			c.state = parseStateComplete

		} else {
			available := data.Len()
			data.Next(available)
			c.valueLength -= uint32(available)
			return io.EOF
		}
	}

	return nil
}

func (c *Command) isComplete() bool {
	return c.state == parseStateComplete
}

func (c *Command) isResponse() bool {
	return c.commandType == RESPONSE
}