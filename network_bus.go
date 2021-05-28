package gfbus

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/rpc"
	"sync"
)

// NetworkBus - object capable of subscribing to remote event buses in addition to remote event
// busses subscribing to it's local event bus. Compoed of a server and client
type NetworkBus struct {
	*Client
	*Server
	service   *NetworkBusService
	sharedBus Bus
	address   string
	path      string
}

// NewNetworkBus - returns a new network bus object at the server address and path
func NewNetworkBus(address, path string) *NetworkBus {
	bus := new(NetworkBus)
	bus.sharedBus = New()
	bus.Server = NewServer(address, path, bus.sharedBus)
	bus.Client = NewClient(address, path, bus.sharedBus)
	bus.service = &NetworkBusService{&sync.WaitGroup{}, false}
	bus.address = address
	bus.path = path
	return bus
}

// EventBus - returns wrapped event bus
func (networkBus *NetworkBus) EventBus() Bus {
	return networkBus.sharedBus
}

// NetworkBusService - object capable of serving the network bus
type NetworkBusService struct {
	wg      *sync.WaitGroup
	started bool
}

// Start - helper method to serve a network bus service
func (networkBus *NetworkBus) Start() error {
	var err, clientErr, serverErr error
	var l net.Listener
	service := networkBus.service
	clientService := networkBus.Client.service
	serverService := networkBus.Server.service
	if !service.started {
		server := rpc.NewServer()
		clientErr = server.RegisterName("ServerService", serverService)
		serverErr = server.RegisterName("ClientService", clientService)
		if serverErr == nil && clientErr == nil {
			server.HandleHTTP(networkBus.path, "/debug"+networkBus.path)
			l, err = net.Listen("tcp", networkBus.address)
			if err != nil {
				_ = fmt.Errorf("listen error: %v", err)
			}
			service.wg.Add(1)
			go func(l net.Listener) {
				err = http.Serve(l, nil)
			}(l)
		} else {
			err = clientErr
		}
	} else {
		err = errors.New("Server bus already started")
	}
	return err
}

// Stop - signal for the service to stop serving
func (networkBus *NetworkBus) Stop() {
	service := networkBus.service
	if service.started {
		service.wg.Done()
		service.started = false
	}
}
