package sample

import (
	"github.com/josvazg/remotize"
	"http"
	"io"
	"math"
	"net"
	"os"
	"rpc"
	"strconv"
	"sync"
)

// We use init() to mark some types and interfaces we wnt to be remotized for us...
func init() {
	// This marks URLStore as remotizable
	remotize.Please(new(URLStore))
	// This marks a type (os.File) defined on a another package (os)
	remotize.Please(new(os.File))
	// This marks an interface (io.ReadWritCloser) defined on another package (io)
	remotize.Please(new(io.ReadWriteCloser))
}

//
//
//     URL STORE SAMPLE SECTION: Remotization of a TYPE in the SAME package (from source code)
//
//

// Some type without interface
type URLStore struct {
	store map[string]string
	mutex sync.Mutex
}

// Creates a new (local) URLStore implementation
func NewURLStore() *URLStore {
	return &URLStore{store: make(map[string]string)}
}

// Get a url from the store
func (s *URLStore) Get(shorturl string) string {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.store[shorturl]
}

// Set a shortUrl to Url mapping in the store
func (s *URLStore) Set(shorturl, url string) bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	_, present := s.store[shorturl]
	if present {
		return false
	}
	s.store[shorturl] = url
	return true
}

// startStorerServer starts a RPC URLStorer server given an implementation
func startStorerServer(us URLStorer) (string, os.Error) {
	// You can also search the service by passing the impleemntation to remotize...
	r := remotize.NewService(us)
	rpc.Register(r)
	rpc.HandleHTTP()
	addr := ":12345"
	l, e := net.Listen("tcp", addr)
	if e != nil {
		return "", e
	}
	go http.Serve(l, nil)
	return "localhost" + addr, nil
}

// getRemoteStorerRef Gets a local reference to a remote URLStorer Service
func getRemoteStorerRef(saddr string) (URLStorer, os.Error) {
	client, e := rpc.DialHTTP("tcp", saddr)
	if e != nil {
		return nil, e
	}
	return remotize.NewRemote(client, new(URLStorer)).(URLStorer), nil
}

//
//
//     CALCER SAMPLE SECTION: Remotization of an INTERFACE in the SAME package (from source code)
//
//

// Some interface
// The end of the comment make it remotizable...
// (remotize)
type Calcer interface {
	Add(float64, float64) float64
	AddTo(*float64, float64)
	Divide(float64, float64) (float64, os.Error)
	Multiply(float64, float64) float64
	Pi() float64
	Randomize()
	RandomizeSeed(float64)
	Subtract(float64, float64) float64
}

// The type implementing it
type Calc struct{}

// Add function returns the addition
func (c *Calc) Add(op1 float64, op2 float64) float64 {
	return op1 + op2
}

// AddTo will place the result back to op1 (pointer)
func (c *Calc) AddTo(op1 *float64, op2 float64) {
	*op1 = *op1 + op2
}

// Subtract return the subtraction
func (c *Calc) Subtract(op1 float64, op2 float64) float64 {
	return op1 - op2
}

// Multiply returns the multiplication
func (c *Calc) Multiply(op1 float64, op2 float64) float64 {
	return op1 * op2
}

// Dividde returns the divide result or an error
func (c *Calc) Divide(op1 float64, op2 float64) (float64, os.Error) {
	if op2 == 0 {
		return 0, os.NewError("Divide " + strconv.Ftoa64(op1, 'f', -1) + " by ZERO!?!")
	}
	return op1 / op2, nil
}

// Pi returns the Pi number
func (c *Calc) Pi() float64 {
	return math.Pi
}

// Randomize does nothing really, its just a rpc WITHOUT any args or results
func (c *Calc) Randomize() {
}

// Randomize seed does nothing either, it's just a sample of rpc WITH 1 arg a 0 results
func (c *Calc) RandomizeSeed(seed float64) {
}

// startCalcerServer starts a RPC Calcer server using Calc as the implementation
func startCalcerServer() (string, os.Error) {
	// You can access the remotized code directly, it should be created by now...
	r := NewCalcerService(new(Calc))
	rpc.Register(r)
	rpc.HandleHTTP()
	addr := ":1234"
	l, e := net.Listen("tcp", addr)
	if e != nil {
		return "", e
	}
	go http.Serve(l, nil)
	return "localhost" + addr, nil
}

// getRemoteCalcerRef returns a local reference to a remote Calcer RPC service
func getRemoteCalcerRef(saddr string) (Calcer, os.Error) {
	client, e := rpc.DialHTTP("tcp", saddr)
	if e != nil {
		return nil, e
	}
	return NewRemoteCalcer(client), nil
}

//
//
//     FILER SAMPLE SECTION: Remotization of a TYPE from OTHER package (from reflection type)
//
//

type FileStore struct {

}

// Create will create a local file and give an address to reach the proper Filer RPC service 
func (fs *FileStore) Create(name string) (string, os.Error) {
	f, e := os.Create(name)
	if e != nil {
		return "", e
	}
	r := NewFilerService(f)
	rpc.Register(r)
	rpc.HandleHTTP()
	l, e := net.Listen("tcp", ":0")
	if e != nil {
		return "", e
	}
	addr := l.Addr().String()
	go http.Serve(l, nil)
	return addr, nil
}

