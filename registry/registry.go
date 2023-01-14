package registry

import (
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

// AnneRegistry is a simple register center,provide following functions
// add a server and receive heartbeat to keep it alive
// returns all alive servers and delete dead servers sync simultaneously

type AnneRegistry struct {
	timeout time.Duration
	mu      sync.Mutex //protect following
	servers map[string]*ServerItem
}

type ServerItem struct {
	Addr  string
	start time.Time
}

const (
	defaultPath    = "/_annerpc_/registry"
	defaultTimeout = time.Minute * 5
)

// New Create a registy instance with timeout setting
func New(timeout time.Duration) *AnneRegistry {
	return &AnneRegistry{
		servers: make(map[string]*ServerItem),
		timeout: timeout,
	}
}

var DefaultAnneRegister = New(defaultTimeout)

func (r *AnneRegistry) putServer(addr string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s := r.servers[addr]
	if s == nil {
		r.servers[addr] = &ServerItem{Addr: addr, start: time.Now()}
	} else {
		s.start = time.Now() //if exists,update start time to keep alive
	}
}

func (r *AnneRegistry) aliveServers() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	var alive []string
	for addr, s := range r.servers {
		if r.timeout == 0 || s.start.Add(r.timeout).After(time.Now()) {
			alive = append(alive, addr)
		} else {
			delete(r.servers, addr)
		}
	}
	sort.Strings(alive)
	return alive
}

// Runs at /_annerpc_/registry
func (a *AnneRegistry) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "GET":
		//server is in req,Header
		w.Header().Set("X-Annerpc-Servers", strings.Join(a.aliveServers(), ","))
	case "POST":
		addr := req.Header.Get("X-Annerpc-Server")
		if addr == "" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		a.putServer(addr)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// HandleHTTP registers an HTTP handler for AnneRegistry messages on registryPath
func (a *AnneRegistry) HandleHTTP(registryPath string) {
	http.Handle(registryPath, a)
	log.Println("rpc registry path:", registryPath)
}

func HandleHTTP() {
	DefaultAnneRegister.HandleHTTP(defaultPath)
}

// Heartbeat send a heartbeat message every once in a while
// it's a helper function for a server to register or send heartbeat
func Heartbeat(registry, addr string, duration time.Duration) {
	if duration == 0 {
		//make sure there is enough time to send heartbeat before
		//it's removed from registry
		duration = defaultTimeout - time.Duration(1)*time.Minute
	}
	var err error
	err = sendHeartbeat(registry, addr)
	go func() {
		t := time.NewTicker(duration)
		for err == nil {
			<-t.C
			err = sendHeartbeat(registry, addr)
		}
	}()
}

func sendHeartbeat(registry, addr string) error {
	log.Println(addr, "send heart beat to registry", registry)
	httpClient := &http.Client{}
	req, _ := http.NewRequest("POST", registry, nil)
	req.Header.Set("X-Annerpc-Server", addr)
	if _, err := httpClient.Do(req); err != nil {
		log.Println("rpc server: heart beat err:", err)
		return err
	}
	return nil
}
