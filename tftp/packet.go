package tftp

import (
	"bytes"
	"encoding/binary"
	"errors"
)

// OpCode represent a TFTP operation code
type OpCode uint16

const (
	OpRRQ   OpCode = 1 // Read Request
	OpWRQ   OpCode = 2 // Write Request
	OpDATA  OpCode = 3 // Data
	OpACK   OpCode = 4 // Acknowledgment
	OpERROR OpCode = 5 // Error
)

type ErrorCode uint16

const (
	ErrorNotDefined           ErrorCode = 0
	ErrorFileNotFount         ErrorCode = 1
	ErrorAccessViolation      ErrorCode = 2
	ErrorDiskFull             ErrorCode = 3
	ErrorIllegalTFTPOperation ErrorCode = 4
	ErrorUnknowTransferID     ErrorCode = 5
	ErrorFileAlreadyExist     ErrorCode = 6
	ErrorNoSuchUser           ErrorCode = 7
)

const (
	MaxPacketSize = 516 // Max TFTP packet size
	MaxDataSize   = 512 // Max Data transferred by DATA packet
)

// Extract the operation code from the buffer.
// The op. code is in the first 2 bytes of the packet.
func extractTheOpCode(bytes []byte) OpCode {
	return OpCode(binary.BigEndian.Uint16(bytes[:2]))
}

// RRQ represent a parsed Read Request packet.
type RRQ struct {
	Filename string
	Mode     string
}

// ParseRRQ takes a raw byte slices (include the opcode) and extract the RRQ data.
func ParseRRQ(b []byte) (*RRQ, error) {
	// The min len of RRQ can be 6 bytes: 2bytes for the opcode, >1 byte filename,
	// >1 byte mode, 2 byte separator \x00
	if len(b) < 6 {
		return nil, errors.New("malformed RRQ packet. Packet is too short")
	}

	fileName, rest, found := bytes.Cut(b[2:], []byte{0})
	if !found {
		return nil, errors.New("malformed RRQ packet. Missing null terminator for filename")
	}

	mode, _, found := bytes.Cut(rest, []byte{0})
	if !found {
		return nil, errors.New("malformed RRQ packet. Missing null terminator for mode")
	}

	return &RRQ{
		Filename: string(fileName),
		Mode:     string(mode),
	}, nil
}

// DATA represent a parsed Data packet.
type DATA struct {
	BlockId uint16
	Payload []byte
}

// Marshal creates slice of bytes (packet) ready to send from a DATA struct
func (pd DATA) Marshal() ([]byte, error) {
	buff := new(bytes.Buffer)

	// Writes the opcode
	err := binary.Write(buff, binary.BigEndian, uint16(OpDATA))
	if err != nil {
		return nil, err
	}

	// Writes the blockId
	err = binary.Write(buff, binary.BigEndian, pd.BlockId)
	if err != nil {
		return nil, err
	}

	buff.Write(pd.Payload)

	return buff.Bytes(), nil
}

// ACK represent a parsed Acknowledge packet.
type ACK struct {
	BlockId uint16
}

// ParseACK takes a raw byte slices (include the opcode) and extract the ACK packet.
func ParseACK(b []byte) (*ACK, error) {
	if len(b) < 4 {
		return nil, errors.New("malformed ACK packet. Packet is too short")
	}
	return &ACK{BlockId: binary.BigEndian.Uint16(b[2:4])}, nil
}

type ERROR struct {
	ErrorCode    ErrorCode
	ErrorMessage string
}

func (e ERROR) Marshal() ([]byte, error) {
	buff := new(bytes.Buffer)

	// Writes the opcode
	err := binary.Write(buff, binary.BigEndian, uint16(OpERROR))
	if err != nil {
		return nil, err
	}

	// Writes the error code
	err = binary.Write(buff, binary.BigEndian, uint16(e.ErrorCode))
	if err != nil {
		return nil, err
	}

	// Write the error message
	buff.WriteString(e.ErrorMessage)

	// Write the end 0-byte
	err = binary.Write(buff, binary.BigEndian, uint8(0))
	if err != nil {
		return nil, err
	}

	return buff.Bytes(), nil
}
