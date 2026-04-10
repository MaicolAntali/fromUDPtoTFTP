package main

import (
	"log"

	"github.com/maicolantali/fromUDPtoTFTP/pkg/tftp"
)

func main() {
	server := tftp.UdpServer{Address: "localhost:69"}

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed: %v\n", err)
	}
}
