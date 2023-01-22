//go:build windows && amd64

package server

import "C"
import (
	"context"
	"github.com/TrueGameover/transaq-grpc/src/client"
	server2 "github.com/TrueGameover/transaq-grpc/src/grpc/server"
	"github.com/rs/zerolog"
	"time"
)

type sendCommand func(msg string) (data string, err error)

func NewConnectService(
	sendCommandFunc *func(msg string) (data string, err error),
	messagesChannel <-chan string,
	clientExists *client.ClientExists,
	logger *zerolog.Logger,
) *ConnectService {
	sendFunc := sendCommand(*sendCommandFunc)
	serverLogger := logger.With().Str("Service", "Server").Logger()

	return &ConnectService{
		txmlSendCommand: &sendFunc,
		messagesChannel: messagesChannel,
		localLogger:     serverLogger,
		clientExists:    clientExists,
	}
}

type ConnectService struct {
	server2.UnimplementedConnectServiceServer

	txmlSendCommand *sendCommand
	messagesChannel <-chan string
	localLogger     zerolog.Logger
	clientExists    *client.ClientExists
	messagesCount   uint
}

func (s *ConnectService) SendCommand(_ context.Context, request *server2.SendCommandRequest) (*server2.SendCommandResponse, error) {
	msg, err := (*s.txmlSendCommand)(request.Message)

	if err != nil {
		s.localLogger.Error().Err(err)
		return nil, err
	}

	return &server2.SendCommandResponse{
		Message: msg,
	}, nil
}

func (s *ConnectService) FetchResponseData(_ *server2.DataRequest, srv server2.ConnectService_FetchResponseDataServer) error {
	s.messagesCount = 0
	s.clientExists.Connected()
	s.localLogger.Info().Msg("Client connected")

	ctx := srv.Context()
	for {
		timeoutCtx, cancel := context.WithTimeout(ctx, time.Minute*1)

		select {
		case msg := <-s.messagesChannel:
			resp := server2.DataResponse{Message: msg}
			err := srv.Send(&resp)
			if err != nil {
				s.localLogger.Error().Err(err).Msg("Sending error")
			}
			s.messagesCount++

		case <-ctx.Done():
			s.localLogger.Warn().Msgf("Loop done %s", ctx.Err())
			s.clientExists.Disconnected()
			cancel()
			return nil

		case <-timeoutCtx.Done():
			s.localLogger.Info().Msgf("Statistic: %d per minute", s.messagesCount)
			s.messagesCount = 0
		}

		cancel()
	}
}
