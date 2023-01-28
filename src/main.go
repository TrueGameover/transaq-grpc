//go:build windows && amd64

package main

import (
	"context"
	"github.com/TrueGameover/transaq-grpc/src/client"
	server2 "github.com/TrueGameover/transaq-grpc/src/grpc/server"
	"github.com/TrueGameover/transaq-grpc/src/server"
	"github.com/TrueGameover/transaq-grpc/src/transaq"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/sys/windows"
	"google.golang.org/grpc"
	"net"
	"os"
	"os/signal"
	"time"
)

const PoolSize = 100

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

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		appLogger.Panic().Err(err)
	}

	srv := grpc.NewServer()
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
