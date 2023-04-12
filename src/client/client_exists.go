package client

import "sync"

type ClientExists struct {
	mutex          *sync.Mutex
	connectedCount uint32
}

func NewClientExists() *ClientExists {
	return &ClientExists{
		mutex: &sync.Mutex{},
	}
}

func (h *ClientExists) Connected() {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.connectedCount++
}

func (h *ClientExists) Disconnected() {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.connectedCount--

	if h.connectedCount < 0 {
		h.connectedCount = 0
	}
}

func (h *ClientExists) IsConnected() bool {
	return h.connectedCount > 0
}
