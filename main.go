package main

import (
	"os"
	"test/tls"
)

func main() {
	tls.HttpsMain()
	go tls.ServerTLSCA()
	tls.ClientTLSCA()
	_, _ = os.Stdin.Read(make([]byte, 1))
}
