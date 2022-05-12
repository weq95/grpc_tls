package tls

import (
	"fmt"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"log"
	"net"
	"net/http"
	"strings"
)

func H2Main() {
	var mux = http.NewServeMux()
	var server = &http.Server{
		Addr:    ":3999",
		Handler: h2c.NewHandler(mux, &http2.Server{}),
	}

	_ = server.ListenAndServe()
}

func HttpsMain() {
	var mux = http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintf(w, "hello")
	})

	_ = http.ListenAndServeTLS(":8085",
		"./tls/pem/server_cert.pem",
		"./tls/pem/server_key.pem",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.ProtoMinor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
				creds, err := credentials.NewServerTLSFromFile("server.crt", "server.key")
				if err != nil {
					log.Fatal(err)
				}

				var grpcServer = grpc.NewServer(grpc.Creds(creds))
				grpcServer.ServeHTTP(w, r)
				return
			}

			mux.ServeHTTP(w, r)
		}))
}

func TlsGrpcServer() {
	creds, err := credentials.NewServerTLSFromFile("server.crt", "server.key")
	if err != nil {
		log.Fatal(err)
	}

	grpcServer := grpc.NewServer(grpc.Creds(creds))
	var listen, err1 = net.Listen("tcp", ":8086")
	if err1 != nil {
		log.Fatal(err1)
	}

	if err = grpcServer.Serve(listen); err != nil {
		log.Fatal(err1)
	}
}
