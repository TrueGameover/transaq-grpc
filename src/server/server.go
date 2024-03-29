//go:build windows && amd64

package server

import "C"
import (
	"context"
	"github.com/TrueGameover/transaq-grpc/src/client"
	server2 "github.com/TrueGameover/transaq-grpc/src/grpc/server"
	"github.com/TrueGameover/transaq-grpc/src/queue"
	"github.com/TrueGameover/transaq-grpc/src/transaq"
	"github.com/rs/zerolog"
	"time"
)

func NewConnectService(
	transaqHandler *transaq.TransaqHandler,
	messagesQueue *queue.FixedQueue[string],
	clientExists *client.ClientExists,
	logger *zerolog.Logger,
) *ConnectService {
	serverLogger := logger.With().Str("Service", "Server").Logger()

	return &ConnectService{
		messagesQueue:  messagesQueue,
		localLogger:    &serverLogger,
		clientExists:   clientExists,
		transaqHandler: transaqHandler,
	}
}

type ConnectService struct {
	server2.UnimplementedConnectServiceServer

	localLogger    *zerolog.Logger
	clientExists   *client.ClientExists
	messagesCount  uint
	transaqHandler *transaq.TransaqHandler
	messagesQueue  *queue.FixedQueue[string]
}

func (s *ConnectService) SendCommand(_ context.Context, request *server2.SendCommandRequest) (*server2.SendCommandResponse, error) {
	msg, code, err := s.transaqHandler.SendCommand(request.Message)
	if err != nil {
		s.localLogger.Error().Err(err)
		return nil, err
	}

	return &server2.SendCommandResponse{
		Message: msg,
		Code:    code,
	}, nil
}

func (s *ConnectService) FetchResponseData(_ *server2.DataRequest, srv server2.ConnectService_FetchResponseDataServer) error {
	s.messagesCount = 0
	s.clientExists.Connected()
	s.localLogger.Info().Msg("Client connected")

	ctx := srv.Context()

	go func() {
		for {
			timeoutCtx, cancel := context.WithTimeout(ctx, time.Minute*1)

			select {
			case <-timeoutCtx.Done():
				s.localLogger.Info().Msgf("Statistic: %d per minute", s.messagesCount)
				s.messagesCount = 0

			case <-ctx.Done():
				cancel()
				return
			}

			cancel()
		}
	}()

	messagesChannel := s.messagesQueue.Fetch(ctx)

	for {
		select {
		case msg, ok := <-messagesChannel:
			if !ok {
				s.localLogger.Error().Msg("messagesChannel was closed")
				return nil
			}

			resp := server2.DataResponse{Message: msg}
			err := srv.Send(&resp)
			if err != nil {
				s.localLogger.Error().Err(err).Msg("Sending error")
			}
			s.messagesCount++

		case <-ctx.Done():
			s.localLogger.Warn().Msgf("Loop done %s", ctx.Err())
			s.clientExists.Disconnected()
			// do not send disconnect command to transaq
			return nil
		}
	}
}
