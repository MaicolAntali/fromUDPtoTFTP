package tftp

import (
	"log"
	"net"
)

type UdpServer struct {
	Address string
}

func (s UdpServer) ListenAndServe() error {
	addr, err := net.ResolveUDPAddr("udp", s.Address)
	if err != nil {
		return err
	}

	conn, err := net.ListenUDP("udp", addr)
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

			s.HandleRRQ(conn, clientAddr, rrq)
		case OpWRQ:
			log.Printf("Received WRQ packet from %s\n", clientAddr)
		case OpACK:
			log.Printf("Received ACK packet from %s\n", clientAddr)
		default:
			log.Printf("Unknow or unsupported op.code!")
		}

	}
}

func (s UdpServer) HandleRRQ(conn *net.UDPConn, addr *net.UDPAddr, rrq *RRQ) {
	log.Printf("Client wants to read file: %s in mode: %s\n", rrq.Filename, rrq.Mode)

	bytes, _ := DATA{
		BlockId: 1,
		Payload: []byte("Hello! This is my first TFTP packet.\n"),
	}.Marshal()

	_, _ = conn.WriteToUDP(bytes, addr)
}
