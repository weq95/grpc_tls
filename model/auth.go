package model

import (
	"context"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

type Authentication struct {
	User     string
	Password string
}

func (a *Authentication) GetRequestMetadata(ctx context.Context, str ...string) (map[string]string, error) {
	return map[string]string{
		"user":     a.User,
		"password": a.Password,
	}, nil
}

func (a *Authentication) RequireTransportSecurity() bool {
	return false
}

func (a *Authentication) Auth(ctx context.Context) error {
	var md, ok = metadata.FromIncomingContext(ctx)
	if !ok {
		return fmt.Errorf("missing credentials")
	}

	var appid, appkey string

	if val, ok := md["login"]; ok {
		appid = val[0]
	}
	if val, ok := md["password"]; ok {
		appkey = val[0]
	}

	if appkey != a.Password || "1" != appid {
		return grpc.Errorf(codes.Unauthenticated, "invalid token")
	}

	return nil
}
