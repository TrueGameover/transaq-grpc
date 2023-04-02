//go:build windows && amd64

package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"github.com/TrueGameover/transaq-grpc/src/client"
	server2 "github.com/TrueGameover/transaq-grpc/src/grpc/server"
	"github.com/TrueGameover/transaq-grpc/src/server"
	"github.com/TrueGameover/transaq-grpc/src/transaq"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/sys/windows"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"net"
	"os"
	"os/signal"
	"time"
)

const PoolSize = 10000

func main() {
	appLogger := configureLogger()
	messagesChannel := make(chan string, PoolSize)
	transaqHandler := transaq.NewTransaqHandler(appLogger, messagesChannel)
	clientExists := client.NewClientExists()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := transaqHandler.Init(ctx, clientExists)
	if err != nil {
		panic(err)
	}
	defer func() {
		if transaqHandler.IsInited() {
			transaqHandler.Release()
		}
	}()

	lis, err := net.Listen("tcp", "0.0.0.0:50051")
	if err != nil {
		appLogger.Panic().Err(err)
	}

	tlsOptions, err := setupTlsConfiguration()
	if err != nil {
		appLogger.Warn().Err(err).Msg("Tls initialization failed. Skipping...")
		tlsOptions = []grpc.ServerOption{}
	}

	srv := grpc.NewServer(tlsOptions...)
	SetupCloseHandler(srv, appLogger, cancel)

	server2.RegisterConnectServiceServer(srv, server.NewConnectService(transaqHandler, messagesChannel, clientExists, appLogger))

	appLogger.Info().Msg("Press CRTL+C to stop the ConnectService...")

	err = srv.Serve(lis)
	if err != nil {
		appLogger.Panic().Err(err)
	}
}

func SetupCloseHandler(srv *grpc.Server, localLogger *zerolog.Logger, appCancelFunc context.CancelFunc) {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, windows.SIGTERM)
	go func() {
		<-c
		localLogger.Warn().Msg("Ctrl+C pressed in Terminal")
		srv.GracefulStop()
		appCancelFunc()
	}()
}

func configureLogger() *zerolog.Logger {
	zeroLogger := log.With().Timestamp().Logger()
	zeroLogger = zeroLogger.Output(
		zerolog.MultiLevelWriter(
			zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}),
	)

	return &zeroLogger
}

func setupTlsConfiguration() ([]grpc.ServerOption, error) {
	var opts []grpc.ServerOption

	rootCa, err := os.ReadFile("certs/rootCA.crt")
	if err != nil {
		return nil, err
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(rootCa) {
		return nil, errors.New("cannot append rootCA to cert pool")
	}

	serverCert, err := tls.LoadX509KeyPair("certs/transaqGrpcServiceServer.crt", "certs/transaqGrpcServiceServer.key")
	if err != nil {
		return nil, err
	}

	tlsConfig := tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientCAs:    certPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		MinVersion:   tls.VersionTLS13,
	}

	opts = append(opts, grpc.Creds(credentials.NewTLS(&tlsConfig)))

	return opts, nil
}
