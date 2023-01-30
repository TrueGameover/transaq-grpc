package transaq

//#include <stdlib.h>
import "C"

import (
	"context"
	"errors"
	"github.com/TrueGameover/transaq-grpc/src/client"
	"github.com/rs/zerolog"
	"golang.org/x/sys/windows"
	"time"
	"unsafe"
)

const dllPath = "txmlconnector64-6.19.2.21.21.dll"

type TransaqHandler struct {
	txmlconnector    *windows.DLL
	procSetCallback  *windows.Proc
	procSendCommand  *windows.Proc
	procFreeMemory   *windows.Proc
	procInitialize   *windows.Proc
	procUnInitialize *windows.Proc
	forMemoryFree    chan *C.char
	messages         chan string
	localLogger      *zerolog.Logger
}

func NewTransaqHandler(logger *zerolog.Logger, messagesChannel chan string) *TransaqHandler {
	forMemoryFree := make(chan *C.char, cap(messagesChannel))
	localLogger := logger.With().Str("Service", "TransaqHandler").Logger()

	return &TransaqHandler{
		forMemoryFree: forMemoryFree,
		localLogger:   &localLogger,
		messages:      messagesChannel,
	}
}

func (h *TransaqHandler) IsInited() bool {
	return h.txmlconnector != nil
}

func (h *TransaqHandler) Init(appContext context.Context, clientExists *client.ClientExists) error {
	dll, err := windows.LoadDLL(dllPath)
	if err != windows.Errno(0) && err != nil {
		h.localLogger.Error().Msgf("load dll failed %d", err)
		return err
	}

	h.txmlconnector = dll
	h.procSetCallback = h.txmlconnector.MustFindProc("SetCallback")
	h.procSendCommand = h.txmlconnector.MustFindProc("SendCommand")
	h.procFreeMemory = h.txmlconnector.MustFindProc("FreeMemory")
	h.procInitialize = h.txmlconnector.MustFindProc("InitializeEx")
	h.procUnInitialize = h.txmlconnector.MustFindProc("UnInitialize")

	initCommandStr := "<init log_path=\"logs\" log_level=\"2\" logfile_lifetime=\"\"/>"
	initCommandPtr := unsafe.Pointer(C.CString(initCommandStr))
	retVal, _, err := h.procInitialize.Call(uintptr(initCommandPtr))
	if err != windows.Errno(0) {
		err = errors.New("Initialize error: " + err.Error())
		h.localLogger.Error().Err(err)
		return err
	}
	if retVal != 0 {
		errorMsg := h.getStringFromCPointer(retVal)
		h.localLogger.Error().Msg(errorMsg)
		return errors.New(errorMsg)
	}

	_, _, err = h.procSetCallback.Call(windows.NewCallback(h.receiveData))
	if err != windows.Errno(0) {
		return errors.New("Set callback fn error: " + err.Error())
	}

	go h.runFreeMemory(appContext)
	go h.runMessagesClearing(appContext, clientExists)

	return nil
}

func (h *TransaqHandler) runFreeMemory(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case cmsg, ok := <-h.forMemoryFree:
			if !ok {
				h.localLogger.Panic().Msg("ForMemoryFree channel was closed")
			}

			_, _, err := h.procFreeMemory.Call(uintptr(unsafe.Pointer(cmsg)))
			if err != windows.Errno(0) {
				h.localLogger.Error().Err(err)
			}
		}
	}
}

func (h *TransaqHandler) runMessagesClearing(ctx context.Context, clientExists *client.ClientExists) {
	for {
		if clientExists.IsConnected() {
			time.Sleep(time.Second * 1)
			continue
		}

		select {
		case <-ctx.Done():
			return
		case _, ok := <-h.messages:
			if !ok {
				h.localLogger.Panic().Msg("messages channel closed")
			}
		}
	}
}

func (h *TransaqHandler) receiveData(cmsg *C.char) uintptr {
	msg := C.GoString(cmsg)

	select {
	case h.messages <- msg:
	default:
		h.localLogger.Warn().Msg("Messages channel overflow")
	}

	select {
	case h.forMemoryFree <- cmsg:
	default:
		// channel can be full, so clearing immediately
		h.getStringFromCPointer(uintptr(unsafe.Pointer(cmsg)))
		h.localLogger.Warn().Msg("memory for free channel overflow")
	}

	ok := true
	return uintptr(unsafe.Pointer(&ok))
}

func (h *TransaqHandler) Disconnect() {
	_, err := h.SendCommand("<command id=\"disconnect\"/>")
	if err != nil {
		h.localLogger.Error().Err(err)
	}
}

func (h *TransaqHandler) Release() {
	h.Disconnect()

	retVal, _, err := h.procUnInitialize.Call()
	if err != windows.Errno(0) {
		h.localLogger.Error().Msgf("dll uninitialized error: %d", err)
	}

	if retVal != 0 {
		msg := h.getStringFromCPointer(retVal)
		h.localLogger.Error().Msg(msg)
	}

	err = h.txmlconnector.Release()
	if err != nil {
		h.localLogger.Error().Err(err)
	}

	h.txmlconnector = nil
}

func (h *TransaqHandler) getStringFromCPointer(pointer uintptr) string {
	if pointer == 0 {
		return ""
	}

	defer func() {
		_, _, err := h.procFreeMemory.Call(pointer)
		if err != windows.Errno(0) {
			h.localLogger.Error().Err(err)
		}
	}()

	//goland:noinspection GoVetUnsafePointer
	cmsg := (*C.char)(unsafe.Pointer(pointer))
	return C.GoString(cmsg)
}

func (h *TransaqHandler) SendCommand(msg string) (string, error) {
	cMsg := C.CString(msg)
	reqData := unsafe.Pointer(cMsg)
	defer C.free(reqData)

	respPtr, _, err := h.procSendCommand.Call(uintptr(reqData))
	if err != windows.Errno(0) {
		return "", errors.New("call error: " + err.Error())
	}

	respData := h.getStringFromCPointer(respPtr)

	return respData, nil
}
