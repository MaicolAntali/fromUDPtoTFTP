package tftp

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"
)

// streamFileToClient sends the passed file to the client.
func streamFileToClient(conn *net.UDPConn, addr *net.UDPAddr, file *os.File) error {
	currentBlockId := uint16(1)
	fileBuffer := make([]byte, MaxDataSize)
	for {
		bytesRead, err := file.Read(fileBuffer)
		if err != nil {
			if err == io.EOF {
				// File is done. We still need to send the final 0-byte packet.
				// This happens when the file size was an exact multiple of 512
				bytesRead = 0
			} else {
				return fmt.Errorf("Error reading file: %v\n", err)
			}
		}

		dataPck := DATA{
			BlockId: currentBlockId,
			Payload: fileBuffer[:bytesRead],
		}

		if err := sendDataWithRetries(conn, addr, dataPck); err != nil {
			return err
		}

		if bytesRead < 512 {
			return nil
		}
		currentBlockId++
	}
}

// streamFileToClient write the passed file to the host.
func streamFileFromClient(conn *net.UDPConn, addr *net.UDPAddr, file *os.File) error {
	currentBlockId := uint16(1)
	for {
		data, err := waitDataWithRetries(conn, addr, currentBlockId)
		if err != nil {
			return err
		}

		_, _ = file.Write(data.Payload)
		_ = sendAck(conn, addr, currentBlockId)

		if len(data.Payload) < MaxDataSize {
			return nil
		}
		currentBlockId++
	}
}

// sendDataWithRetries sends data a packet; handles the expected ack packet and,
// in case of missing or timeout retries to send the packet.
func sendDataWithRetries(conn *net.UDPConn, addr *net.UDPAddr, data DATA) error {
	maxRetries := 3

	for attempts := 0; attempts < maxRetries; attempts++ {
		if err := sendData(conn, addr, data); err != nil {
			return err
		}

		// 2. Call your new waitForACK() helper
		err := waitForACK(conn, addr, data.BlockId)
		// 3. Evaluate the result
		if err != nil {
			if errors.Is(err, os.ErrDeadlineExceeded) {
				log.Printf("Timeout, retrying...")
				continue // Try again!
			}
			return err // A real network error
		}

		return nil // Success!
	}

	return errors.New("transfer failed: max retries exceeded")
}

func waitDataWithRetries(conn *net.UDPConn, addr *net.UDPAddr, blockId uint16) (*DATA, error) {
	maxRetries := 3

	for attempts := 0; attempts < maxRetries; attempts++ {
		data, err := waitForData(conn, addr, blockId)
		if err != nil {
			if errors.Is(err, os.ErrDeadlineExceeded) {
				log.Printf("Timeout, retrying...")
				sendAck(conn, addr, blockId-1) // Send the prev block id ack
				continue                       // Try again!
			}
			return nil, err
		}

		return data, nil // Success!
	}

	return nil, errors.New("transfer failed: max retries exceeded")
}

// waitForACK wait until a valid ACK packet is received or give up after 3 seconds timeout.
func waitForACK(conn *net.UDPConn, addr *net.UDPAddr, blockID uint16) error {
	ackBuff := make([]byte, MaxPacketSize)

	for {
		if err := conn.SetReadDeadline(time.Now().Add(3 * time.Second)); err != nil {
			return fmt.Errorf("Error setting read timeout for %v. Err: %v\n", addr, err)
		}

		n, respAddr, err := conn.ReadFromUDP(ackBuff)
		if err != nil {
			return err
		}

		if extractTheOpCode(ackBuff[:n]) == OpACK {
			ack, err := ParseACK(ackBuff[:n])
			if err != nil {
				return fmt.Errorf("Error parking the ACK packet: %v\n", err)
			}

			if addr.Port != respAddr.Port {
				if err := sendErrorPacket(conn, respAddr, ErrorUnknownTransferID, "Unknown transfer ID!"); err != nil {
					return err
				}
				continue
			}

			if ack.BlockId != blockID {
				continue
			}

			return nil // ACK received correctly
		}

		continue
	}
}

// waitForData wait until a valid DATA packet is received or give up after 3 seconds timeout.
func waitForData(conn *net.UDPConn, addr *net.UDPAddr, blockID uint16) (*DATA, error) {
	ackBuff := make([]byte, MaxPacketSize)

	for {
		if err := conn.SetReadDeadline(time.Now().Add(3 * time.Second)); err != nil {
			return nil, fmt.Errorf("Error setting read timeout for %v. Err: %v\n", addr, err)
		}

		n, respAddr, err := conn.ReadFromUDP(ackBuff)
		if err != nil {
			return nil, err
		}

		if extractTheOpCode(ackBuff[:n]) == OpDATA {
			data, err := ParseData(ackBuff[:n])
			if err != nil {
				return nil, fmt.Errorf("Error parsing the DATA packet: %v\n", err)
			}

			if addr.Port != respAddr.Port {
				if err := sendErrorPacket(conn, respAddr, ErrorUnknownTransferID, "Unknown transfer ID!"); err != nil {
					return nil, err
				}
				continue
			}

			if data.BlockId != blockID {
				continue
			}

			return data, nil
		}

		continue
	}
}

// sendData send DATA packet.
func sendData(conn *net.UDPConn, addr *net.UDPAddr, data DATA) error {
	bytesToSend, err := data.Marshal()
	if err != nil {
		return fmt.Errorf("Error marshaling data for %v. Err: %v\n", addr, err)
	}

	if _, err := conn.WriteToUDP(bytesToSend, addr); err != nil {
		return err
	}

	return nil
}

// sendAck send ACK packet.
func sendAck(conn *net.UDPConn, addr *net.UDPAddr, blockId uint16) error {
	ack := ACK{BlockId: blockId}

	bytesToSend, err := ack.Marshal()
	if err != nil {
		return fmt.Errorf("Error marshaling ack for %v. Err: %v\n", addr, err)
	}

	if _, err := conn.WriteToUDP(bytesToSend, addr); err != nil {
		return err
	}

	return nil
}

// Send an error packet with the specified error code and message.
func sendErrorPacket(conn *net.UDPConn, addr *net.UDPAddr, code ErrorCode, message string) error {
	errPacket := ERROR{
		ErrorCode:    code,
		ErrorMessage: message,
	}

	bytes, err := errPacket.Marshal()
	if err != nil {
		return fmt.Errorf("Error marshaling the error packer %v\n", errPacket)
	}

	_, err = conn.WriteToUDP(bytes, addr)
	if err != nil {
		return fmt.Errorf("Error sending the error packer to %v\n", addr)
	}

	return nil
}
