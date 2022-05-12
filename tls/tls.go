package tls

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"grpc_tls/model"
	"grpc_tls/proto/src"
	"io/ioutil"
	"log"
	"net"
	"time"
)

func ServerTLS() {
	var creds, err = credentials.NewServerTLSFromFile("./tls/server.crt", "./tls/server.key")
	if err != nil {
		log.Fatal(err)
	}

	var server = grpc.NewServer(grpc.Creds(creds))
	var listen, lErr = net.Listen("tcp", ":5000")
	if lErr != nil {
		log.Fatal(err)
	}

	if err = server.Serve(listen); err != nil {
		log.Fatal("server start err: ", err)
	}
}

func ClientTLS() {
	var creds, err = credentials.NewClientTLSFromFile("./tls/config/server.crt", "weq.com")
	if err != nil {
		log.Fatal(err)
	}

	var conn, cErr = grpc.Dial("localhost:5000", grpc.WithTransportCredentials(creds))
	if cErr != nil {
		log.Fatal(err)
	}

	defer conn.Close()
}

func ClientTLSCA() {
	var ca, cert = loadCa("./tls/pem/client_cert.pem", "./tls/pem/client_key.pem")
	var conn, cErr = grpc.Dial("localhost:5000",
		grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
			Certificates:       []tls.Certificate{cert},
			ServerName:         "127.0.0.1",
			RootCAs:            ca,
			InsecureSkipVerify: true, //因为证书是自己颁发的, 所以这里不能检测,否则会报错
		})), grpc.WithPerRPCCredentials(&model.Authentication{}))
	if cErr != nil {
		log.Fatal(cErr)
	}

	defer conn.Close()

	fmt.Println("tls client success!", time.Now().String())

	var client = src.NewSearchServiceClient(conn)
	var response, rRrr = client.Search(context.Background(), &src.SearchRequest{
		Query: "id=12&name=iphone",
		Page:  1,
		Per:   15,
	})

	if rRrr != nil {
		log.Fatal("查询商品信息错误, 请稍后重试! ", rRrr)
	}

	var betData, _ = json.Marshal(response)
	fmt.Println("商品信息: ", string(betData))
}

func loadCa(certPath, keyPath string) (*x509.CertPool, tls.Certificate) {
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		log.Fatalf("failed to load key pair: %s", err)
	}

	var caFilePath = "./tls/pem/client_ca_cert.pem"
	caBytes, err := ioutil.ReadFile(caFilePath)
	if err != nil {
		log.Fatalf("failed to read ca cert %q: %v", caFilePath, err)
	}
	ca := x509.NewCertPool()
	if ok := ca.AppendCertsFromPEM(caBytes); !ok {
		log.Fatalf("failed to parse %q", caFilePath)
	}

	return ca, cert
}

func ServerTLSCA() {
	var ca, cert = loadCa("./tls/pem/server_cert.pem", "./tls/pem/server_key.pem")
	var server = grpc.NewServer(grpc.Creds(credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    ca,
	})), grpc.UnaryInterceptor(filter))

	//注册服务
	src.RegisterSearchServiceServer(server, new(Product))

	var listen, lErr = net.Listen("tcp", ":5000")
	if lErr != nil {
		log.Fatal("rpc net.Listen err: ", lErr)
	}

	if err := server.Serve(listen); err != nil {
		log.Fatal("rpc server start err: ", lErr)
	}
}

//filter 拦截器
func filter(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()

	return handler(ctx, req)
}

type Product struct {
	auth *model.Authentication
}

func (p *Product) SomeMethod(ctx context.Context, in *src.SearchRequest) (*src.SearchResponse, error) {
	if err := p.auth.Auth(ctx); err != nil {
		return nil, err
	}

	return &src.SearchResponse{
		Code: 200,
		Msg:  "success! \r\n Hello " + in.GetQuery(),
	}, nil
}

// Search Search(context.Context, *SearchRequest) (*SearchRequest, error)
func (p *Product) Search(ctx context.Context, request *src.SearchRequest) (*src.SearchResponse, error) {
	var bteData, _ = json.Marshal(request)
	fmt.Println("client request: ", string(bteData))
	return &src.SearchResponse{
		Code: 200,
		Msg:  "请求成功!",
	}, nil
}
