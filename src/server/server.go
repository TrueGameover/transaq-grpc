//go:build windows && amd64

package server

import "C"
import (
	"context"
	"errors"
	"github.com/TrueGameover/transaq-grpc/src/client"
	server2 "github.com/TrueGameover/transaq-grpc/src/grpc/server"
	"github.com/rs/zerolog"
	"time"
)

type sendCommand func(msg string) (data *string, err error)

func NewConnectService(
	sendCommandFunc *func(msg string) (data *string, err error),
	messagesChannel <-chan string,
	clientExists *client.ClientExists,
	logger *zerolog.Logger,
) *ConnectService {
	sendFunc := sendCommand(*sendCommandFunc)

	return &ConnectService{
		txmlSendCommand: &sendFunc,
		messagesChannel: messagesChannel,
		localLogger:     logger,
		clientExists:    clientExists,
	}
}

type ConnectService struct {
	server2.UnimplementedConnectServiceServer

	txmlSendCommand *sendCommand
	messagesChannel <-chan string
	localLogger     *zerolog.Logger
	clientExists    *client.ClientExists
}

func (s *ConnectService) SendCommand(_ context.Context, request *server2.SendCommandRequest) (*server2.SendCommandResponse, error) {
	msg, _ := (*s.txmlSendCommand)(request.Message)
	if msg == nil {
		return nil, errors.New("nil response")
	}

	return &server2.SendCommandResponse{
		Message: *msg,
	}, nil
}

func (s *ConnectService) FetchResponseData(_ *server2.DataRequest, srv server2.ConnectService_FetchResponseDataServer) error {
	s.clientExists.Connected()
	s.localLogger.Info().Msg("Client connected")

	ctx := srv.Context()
	for {
		timeoutCtx, cancel := context.WithTimeout(ctx, time.Second*5)

		select {
		case msg := <-s.messagesChannel:
			resp := server2.DataResponse{Message: msg}
			err := srv.Send(&resp)
			if err != nil {
				s.localLogger.Error().Err(err).Msg("Sending error")
			}
		case <-ctx.Done():
			s.localLogger.Warn().Msgf("Loop done %s", ctx.Err())
			s.clientExists.Disconnected()
			cancel()
			return nil
		case <-timeoutCtx.Done():
			s.localLogger.Info().Msg("no message received")
		}

		cancel()
	}
}
