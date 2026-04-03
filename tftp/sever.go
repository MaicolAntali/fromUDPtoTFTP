package tftp

import (
	"errors"
	"log"
	"net"
	"os"
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
			log.Printf("Received RWRQ packet from %s\n", clientAddr)
			rrq, err := ParseRWRQ(rawPacket[:n])
			if err != nil {
				log.Printf("Malformed RWRQ request from %s, error: %v\n", clientAddr, err)
				continue
			}

			go s.HandleRRQ(clientAddr, rrq)
		case OpWRQ:
			log.Printf("Received WRQ packet from %s\n", clientAddr)
			rrq, err := ParseRWRQ(rawPacket[:n])
			if err != nil {
				log.Printf("Malformed RWRQ request from %s, error: %v\n", clientAddr, err)
				continue
			}

			go s.HandleWRQ(clientAddr, rrq)
		default:
			if err := sendErrorPacket(conn, clientAddr, ErrorIllegalTFTPOperation, "Illegal TFTP operation"); err != nil {
				log.Println(err)
			}
		}

	}
}

func (s UdpServer) HandleRRQ(addr *net.UDPAddr, rrq *RWRQ) {
	log.Printf("Client wants to read file: %s in mode: %s\n", rrq.Filename, rrq.Mode)

	conn, err := createUdpConn(":0")
	if err != nil {
		log.Println("Impossible creare a new UDP socket", err)
		return
	}
	defer conn.Close()

	// Open the file
	file, err := os.Open(rrq.Filename)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if err := sendErrorPacket(conn, addr, ErrorFileNotFount, "File not found!"); err != nil {
				log.Printf("Error sending the error packet: %v\n", err)
			}
		}
		log.Printf("Error opening file %s: %s\n\n", rrq.Filename, err)
		return
	}
	defer file.Close()

	if err := streamFileToClient(conn, addr, file); err != nil {
		log.Printf("Error streaming the file %v to %v: %v\n", file, addr, err)
		return
	}
	log.Printf("File %v Transfer completed to %v!\n", file, addr)
}

func (s UdpServer) HandleWRQ(addr *net.UDPAddr, rrq *RWRQ) {
	log.Printf("Client wants to write file: %s in mode: %s\n", rrq.Filename, rrq.Mode)

	conn, err := createUdpConn(":0")
	if err != nil {
		log.Println("Impossible creare a new UDP socket", err)
		return
	}
	defer conn.Close()

	// Create the file
	file, err := os.Create(rrq.Filename)
	if err != nil {
		log.Printf("Error opening file %s: %s\n\n", rrq.Filename, err)
		return
	}
	defer file.Close()

	_ = sendAck(conn, addr, 0) // Special block ID for WRQ

	if err := streamFileFromClient(conn, addr, file); err != nil {
		log.Printf("Error streaming the file %v to %v: %v\n", file, addr, err)
		return
	}
	log.Printf("File %v Transfer completed to %v!\n", file, addr)
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
