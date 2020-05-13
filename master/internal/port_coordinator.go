package internal

import (
	"fmt"
	"strconv"

	"github.com/determined-ai/determined/master/pkg/actor/api"
	"github.com/gorilla/websocket"

	"github.com/determined-ai/determined/master/pkg/actor"
)

type portRequest struct {
	portName string
	socket   *websocket.Conn
}

type portSet struct {
	portName string
	port int
	socket   *websocket.Conn
}

type resetPorts struct {}

type portMetadata struct {
	port int
	isSet bool
	requestsWaiting []*actor.Ref
}

type portCoordinator struct {
	portRequests map[string]*portMetadata
	numRequests int
}

func newPortCoordinator() actor.Actor {
	return &portCoordinator{
		portRequests: make(map[string]*portMetadata),
		numRequests: 0,
	}
}

func (p *portCoordinator) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case portRequest:
		if err := p.processPortRequest(ctx, msg); err != nil {
			return err
		}

	case portSet:
		if err := p.processPortSet(ctx, msg); err != nil {
			return err
		}

	case resetPorts:
		p.portRequests = make(map[string]*portMetadata)

	default:
		break
	}
	return nil
}


func (p *portCoordinator) processPortSet(ctx *actor.Context, msg portSet) error {
	p.initSocketActor(ctx, msg.socket)

	if _, ok := p.portRequests[msg.portName]; !ok {
		p.portRequests[msg.portName] = &portMetadata{
			requestsWaiting: make([]*actor.Ref, 0),
		}
	}

	if p.portRequests[msg.portName].isSet {
		ctx.Log().Errorf("Port for %s is already set at %d", msg.portName, p.portRequests[msg.portName].port)
	}

	p.portRequests[msg.portName].port = msg.port
	p.portRequests[msg.portName].isSet = true

	for _, ref := range p.portRequests[msg.portName].requestsWaiting {
		if err := api.WriteSocketRaw(ctx, ref, strconv.Itoa(p.portRequests[msg.portName].port)); err != nil {
			ctx.Log().WithError(err).Errorf("cannot write to socket")
		}
	}

	p.portRequests[msg.portName].requestsWaiting = make([]*actor.Ref, 0)

	return nil
}


func (p *portCoordinator) processPortRequest(ctx *actor.Context, msg portRequest) error {
	ref := p.initSocketActor(ctx, msg.socket)

	if _, ok := p.portRequests[msg.portName]; !ok {
		p.portRequests[msg.portName] = &portMetadata{
			requestsWaiting: make([]*actor.Ref, 0),
		}
	}

	if p.portRequests[msg.portName].isSet {
		if err := api.WriteSocketRaw(ctx, ref, strconv.Itoa(p.portRequests[msg.portName].port)); err != nil {
			ctx.Log().WithError(err).Errorf("cannot write to socket")
		}
	} else {
		p.portRequests[msg.portName].requestsWaiting = append(p.portRequests[msg.portName].requestsWaiting, ref)
	}

	return nil
}


func (p *portCoordinator) initSocketActor(ctx *actor.Context, socket *websocket.Conn) *actor.Ref {
	a := api.WrapSocket(socket, nil, false)
	ref, _ := ctx.ActorOf(fmt.Sprintf("portRequest-socket-%d", p.numRequests), a)
	ctx.Respond(ref)
	p.numRequests++

	return ref
}




