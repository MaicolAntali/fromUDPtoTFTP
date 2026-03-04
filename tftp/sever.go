package tftp

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"time"
)

type UdpServer struct {
	Address string
}

func (s UdpServer) ListenAndServe() error {
	conn, err := createUdpConn(s.Address)
	if err != nil {
		return err
	}
	defer conn.Close()

	rawPacket := make([]byte, MaxPacketSize)
	for {
		n, clientAddr, err := conn.ReadFromUDP(rawPacket)
		if err != nil {
			log.Printf("Error reading from UDP %s: %v\n", clientAddr, err)
			continue
		}

		if n < 2 {
			log.Printf("Packet too short from %s\n", clientAddr)
			continue
		}

		switch extractTheOpCode(rawPacket[:n]) {
		case OpRRQ:
			log.Printf("Received RRQ packet from %s\n", clientAddr)
			rrq, err := ParseRRQ(rawPacket[:n])
			if err != nil {
				log.Printf("Malformed RRQ request from %s, error: %v\n", clientAddr, err)
				continue
			}

			go s.HandleRRQ(clientAddr, rrq)
		case OpWRQ:
			log.Printf("Received WRQ packet from %s\n", clientAddr)
		case OpACK:
			log.Printf("Received ACK packet from %s\n", clientAddr)
		default:
			log.Printf("Unknow or unsupported op.code!")
		}

	}
}

func (s UdpServer) HandleRRQ(addr *net.UDPAddr, rrq *RRQ) {
	log.Printf("Client wants to read file: %s in mode: %s\n", rrq.Filename, rrq.Mode)

	conn, err := createUdpConn(":0")
	if err != nil {
		log.Println("Impossible creare a new UDP socket", err)
		return
	}
	defer conn.Close()

	currentBlockId := uint16(1)
	payload := []byte("Hello! This is my first TFTP packet.\n")

	for {
		dataPck := DATA{
			BlockId: currentBlockId,
			Payload: payload,
		}

		if err := sendDataAndWaitForAck(conn, addr, dataPck); err != nil {
			log.Printf("Aborting transfer to %s: %v\n", addr, err)
			return
		}

		log.Printf("Successfully sent block %d to %v\n", currentBlockId, addr)
		if len(payload) < 512 {
			log.Println("Transfer completed!")
			return
		}

		currentBlockId++

	}
}

// createUdpConn creates a new UDP socket
func createUdpConn(address string) (*net.UDPConn, error) {
	addr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, err
	}

	return conn, err
}

// sendDataAndWaitForAck handles the messy network I/O, timeouts, and retries.
// It returns nil if the client successfully ACK'd the exact block we sent.
func sendDataAndWaitForAck(conn *net.UDPConn, clientAddr *net.UDPAddr, data DATA) error {
	bytesToSend, err := data.Marshal()
	if err != nil {
		return fmt.Errorf("Error marshaling data for %v. Err: %v\n", clientAddr, err)
	}

	ackBuff := make([]byte, MaxPacketSize)
	maxRetries := 3

	for attempts := 0; attempts < maxRetries; attempts++ {
		_, err := conn.WriteToUDP(bytesToSend, clientAddr)
		if err != nil {
			return fmt.Errorf("Error sending data to %v. Err: %v\n", clientAddr, err)
		}

		if conn.SetReadDeadline(time.Now().Add(3*time.Second)) != nil {
			return fmt.Errorf("Error setting read timeout for %v. Err: %v\n", clientAddr, err)
		}
		n, addr, err := conn.ReadFromUDP(ackBuff)
		if err != nil {
			if errors.Is(err, os.ErrDeadlineExceeded) {
				log.Printf("Timeout waiting for ACK %d, retrying (%d/%d)...\n", data.BlockId, attempts+1, maxRetries)
				continue
			}
			return err
		}

		if extractTheOpCode(ackBuff[:n]) == OpACK {
			ack, err := ParseACK(ackBuff[:n])
			if err == nil && ack.BlockId == data.BlockId && clientAddr.Port == addr.Port {
				return nil // SUCCESS! The exact block was acknowledged.
			}
		}
	}

	return errors.New("transfer failed: max retries exceeded")
}
