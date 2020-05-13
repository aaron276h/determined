package internal

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
	"fmt"

	"github.com/gorilla/websocket"
	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/actor"
)

type systemForPortCoordinator struct {
	system *actor.System
	t      *testing.T
}

var upgrader = websocket.Upgrader{}

func (s *systemForPortCoordinator) requestHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.t.Errorf("Error: %s", err)
	}

	portName := r.URL.Path
	query := r.URL.Query()

	trialIdString := query["trial_id"]
	trialID, err := strconv.Atoi(trialIdString[0])
	if err != nil {
		s.t.Errorf("Error: %s", err)
	}

	setPortString:= query["set_port"]
	var socketActor actor.Response
	if strings.EqualFold(setPortString[0], "True") {
		portString := query["port"]
		port, err := strconv.Atoi(portString[0])
		if err != nil {
			s.t.Errorf("Error: %s", err)
		}

		socketActor = s.system.AskAt(actor.Addr(fmt.Sprintf("trial-%d-portCoordinator", trialID)),
			portSet{portName, port, conn})

	} else {
		socketActor = s.system.AskAt(actor.Addr(fmt.Sprintf("trial-%d-portCoordinator", trialID)),
			portRequest{portName, conn})
	}

	actorRef := socketActor.Get().(*actor.Ref)

	// Wait for the websocket actor to terminate.
	if err := actorRef.AwaitTermination(); err != nil {
		s.t.Logf("Server socket closed")
	}
}

func requestPort(t *testing.T, addr string, wg *sync.WaitGroup, portName string, trialID int, expectedPort int) {
	defer wg.Done()
	u := url.URL{
		Scheme:   "ws",
		Host:     addr,
		Path:     fmt.Sprintf("/ws/port-coordinator/%s", portName),
		RawQuery: fmt.Sprintf("set_port=False&trial_id=%d", trialID),
	}
	c, resp, err := websocket.DefaultDialer.Dial(u.String(), nil)
	assert.NilError(t, err)
	defer func() {
		resp.Close = true
		if errClose := c.Close(); errClose != nil {
			t.Logf("Error closing socket: %s", errClose)
		}
	}()

	_, message, err := c.ReadMessage()
	assert.NilError(t, err)
	assert.Equal(t, string(message),
		string(expectedPort), "Did not receive `%d` "+
			"response from server, got instead: %s", expectedPort, string(message))
}

func setPort(
	t *testing.T,
	addr string,
	wg *sync.WaitGroup,
	portName string,
	trialID,
	port int,
) {
	defer wg.Done()
	u := url.URL{
		Scheme:   "ws",
		Host:     addr,
		Path:     fmt.Sprintf("/ws/port-coordinator/%s", portName),
		RawQuery: fmt.Sprintf("set_port=True&trial_id=%d&port=%d", trialID, port),
	}
	c, resp, err := websocket.DefaultDialer.Dial(u.String(), nil)
	assert.NilError(t, err)
	defer func() {
		resp.Close = true
		if errClose := c.Close(); errClose != nil {
			t.Logf("Error closing socket: %s", errClose)
		}
	}()
}

func TestPortCoordinatorLayer(t *testing.T) {
	addr := "localhost:8080"
	numWorkers := 8
	port := 12345
	trialId := 23
	portName := "best_port"
	var wg sync.WaitGroup

	system := actor.NewSystem("")
	portCoordinator := newPortCoordinator()
	_, created := system.ActorOf(actor.Addr(fmt.Sprintf("trial-%d-portCoordinator", trialId)), portCoordinator)
	if !created {
		t.Fatal("unable to create port coordinator")
	}
	systemRef := &systemForPortCoordinator{system, t}

	serverMutex := http.NewServeMux()
	server := http.Server{Addr: addr, Handler: serverMutex}
	serverMutex.HandleFunc("/ws/port-coordinator/*", systemRef.requestHandler)

	go func() {
		if err := server.ListenAndServe(); err != nil {
			t.Logf("port coordinator server stopped.")
		}
	}()

	// Wait for server to start up.
	time.Sleep(2 * time.Second)

	wg.Add(numWorkers)
	go setPort(t, addr, &wg, portName, trialId, port)

	for i := 1; i < numWorkers; i++ {
		go requestPort(t, addr, &wg, portName, trialId, port)
	}
	wg.Wait()
}
