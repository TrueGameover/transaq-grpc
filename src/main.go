//go:build windows && amd64

package main

import "C"
import (
	"context"
	"errors"
	"github.com/TrueGameover/transaq-grpc/src/client"
	server2 "github.com/TrueGameover/transaq-grpc/src/grpc/server"
	"github.com/TrueGameover/transaq-grpc/src/server"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/sys/windows"
	"google.golang.org/grpc"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
	"unsafe"
)

//#include <stdlib.h>
import "C"

const DllPath = "txmlconnector64-6.19.2.21.21.dll"
const PoolSize = 100

var (
	txmlconnector    *windows.DLL
	procSetCallback  *windows.Proc
	procSendCommand  *windows.Proc
	procFreeMemory   *windows.Proc
	procInitialize   *windows.Proc
	procUnInitialize *windows.Proc
)

var (
	Messages      = make(chan string, PoolSize)
	ForMemoryFree = make(chan *C.char, PoolSize)
)

func main() {
	appLogger := configureLogger()

	defer func() {
		_, _, err := procUnInitialize.Call()
		if err != nil {
			appLogger.Error().Err(err)
		}
	}()

	err := initLibrary()
	if err != nil {
		appLogger.Panic().Err(err)
	}

	appContext, cancel := context.WithCancel(context.Background())
	defer cancel()

	clientExists := client.NewClientExists()

	go runFreeMemory(appContext)
	go runMessagesClearing(appContext, clientExists, appLogger)

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		appLogger.Panic().Err(err)
	}

	srv := grpc.NewServer()
	SetupCloseHandler(srv, appLogger, cancel)

	sendCommand := TxmlSendCommand
	server2.RegisterConnectServiceServer(srv, server.NewConnectService(&sendCommand, Messages, clientExists, appLogger))

	appLogger.Info().Msg("Press CRTL+C to stop the ConnectService...")

	err = srv.Serve(lis)
	if err != nil {
	}
}

func initLibrary() error {
	dll, err := windows.LoadDLL(DllPath)
	if err != nil {
		return err
	}
	txmlconnector = dll

	procSetCallback = txmlconnector.MustFindProc("SetCallback")
	procSendCommand = txmlconnector.MustFindProc("SendCommand")
	procFreeMemory = txmlconnector.MustFindProc("FreeMemory")
	procInitialize = txmlconnector.MustFindProc("Initialize")
	procUnInitialize = txmlconnector.MustFindProc("UnInitialize")

	logPathPtr := uintptr(unsafe.Pointer(C.CString("logs")))
	logLevelPtr := uintptr(2)
	_, _, err = procInitialize.Call(logPathPtr, logLevelPtr)
	if err != windows.Errno(0) {
		return errors.New("Initialize error: " + err.Error())
	}

	_, _, err = procSetCallback.Call(windows.NewCallback(receiveData))
	if err != windows.Errno(0) {
		return errors.New("Set callback fn error: " + err.Error())
	}

	return nil
}

//export receiveData
func receiveData(cmsg *C.char) uintptr {
	msg := C.GoString(cmsg)

	Messages <- msg
	ForMemoryFree <- cmsg

	ok := true
	return uintptr(unsafe.Pointer(&ok))
}

func TxmlSendCommand(msg string) (data *string, err error) {
	reqData := C.CString(msg)
	reqDataPtr := uintptr(unsafe.Pointer(reqData))
	respPtr, _, err := procSendCommand.Call(reqDataPtr)
	if err != syscall.Errno(0) {
		//log.Error("txmlSendCommand() ", err.Error())
		return nil, errors.New("call error")
	}

	//goland:noinspection GoVetUnsafePointer
	cmsg := (*C.char)(unsafe.Pointer(respPtr))
	respData := C.GoString((*C.char)(cmsg))

	ForMemoryFree <- cmsg

	return &respData, nil
}

func runFreeMemory(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case cmsg, ok := <-ForMemoryFree:
			if !ok {
				panic("ForMemoryFree channel was closed")
			}

			_, _, err := procFreeMemory.Call(uintptr(unsafe.Pointer(cmsg)))
			if err != windows.Errno(0) {
			}
		}
	}
}

func runMessagesClearing(ctx context.Context, clientExists *client.ClientExists, localLogger *zerolog.Logger) {
	for {
		if clientExists.IsConnected() {
			time.Sleep(time.Second * 1)
			continue
		}

		select {
		case <-ctx.Done():
			return
		case _, ok := <-Messages:
			if !ok {
				localLogger.Panic().Msg("messages channel closed")
			}
		}
	}
}

func SetupCloseHandler(srv *grpc.Server, localLogger *zerolog.Logger, appCancelFunc context.CancelFunc) {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		localLogger.Warn().Msg("Ctrl+C pressed in Terminal")
		srv.GracefulStop()
		close(Messages)
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
