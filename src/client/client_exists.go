package client

import "sync"

type ClientExists struct {
	mutex     *sync.Mutex
	connected bool
}

func NewClientExists() *ClientExists {
	return &ClientExists{
		mutex: &sync.Mutex{},
	}
}

func (h *ClientExists) Connected() {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.connected = true
}

func (h *ClientExists) Disconnected() {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.connected = false
}

func (h *ClientExists) IsConnected() bool {
	return h.connected
}
