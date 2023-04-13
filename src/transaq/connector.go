package transaq

//#include <stdlib.h>
import "C"

import (
	"context"
	"errors"
	"github.com/TrueGameover/transaq-grpc/src/client"
	"github.com/TrueGameover/transaq-grpc/src/queue"
	"github.com/rs/zerolog"
	"golang.org/x/sys/windows"
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
	messagesQueue    *queue.FixedQueue[string]
	localLogger      *zerolog.Logger
}

func NewTransaqHandler(logger *zerolog.Logger, messagesQueue *queue.FixedQueue[string]) *TransaqHandler {
	forMemoryFree := make(chan *C.char, messagesQueue.GetMaxSize())
	localLogger := logger.With().Str("Service", "TransaqHandler").Logger()

	return &TransaqHandler{
		forMemoryFree: forMemoryFree,
		localLogger:   &localLogger,
		messagesQueue: messagesQueue,
	}
}

func (h *TransaqHandler) IsInited() bool {
	return h.txmlconnector != nil
}

func (h *TransaqHandler) Init(appContext context.Context, _ *client.ClientExists) error {
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

func (h *TransaqHandler) receiveData(cmsg *C.char) uintptr {
	msg := C.GoString(cmsg)

	h.messagesQueue.Push(msg)

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
	_, _, err := h.SendCommand("<command id=\"disconnect\"/>")
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

func (h *TransaqHandler) SendCommand(msg string) (string, uint64, error) {
	cMsg := C.CString(msg)
	reqData := unsafe.Pointer(cMsg)
	defer C.free(reqData)

	respPtr, _, err := h.procSendCommand.Call(uintptr(reqData))
	respData := h.getStringFromCPointer(respPtr)

	h.localLogger.Info().Msg(respData)
	if err != windows.Errno(0) {
		windowsError, _ := err.(windows.Errno)
		h.localLogger.Error().Err(err).Msgf("call error with response ( %d )", uint64(windowsError))
	}

	if len(respData) > 0 {
		windowsError, _ := err.(windows.Errno)
		return respData, uint64(windowsError), nil
	}

	if err != windows.Errno(0) {
		windowsError, _ := err.(windows.Errno)
		return "", uint64(windowsError), errors.New("call error: " + err.Error())
	}

	return respData, 0, nil
}
