package main

import (
	"grpc_tls/tls"
	"os"
)

func main() {
	go tls.ServerTLSCA()
	tls.ClientTLSCA()
	_, _ = os.Stdin.Read(make([]byte, 1))
}
