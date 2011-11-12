package sample

import (
	"github.com/josvazg/remotize"
	"io"
	"math"
	"os"
	"strconv"
	"sync"
)

func init() {
	// This marks URLStore as remotizable
	remotize.Please(new(URLStore))
    // This marks a type (os.File) defined on a another package (os)
	remotize.Please(new(os.File))
	// This marks an interface (io.ReadWritCloser) defined on another package (io)
	remotize.Please(new(io.ReadWriteCloser))
}

// Some type without interface
type URLStore struct {
	store map[string]string
	mutex sync.Mutex
}

func NewURLStore() *URLStore {
	return &URLStore{store:make(map[string]string)}
}

func (s *URLStore) Get(key string) string {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.store[key]
}

func (s *URLStore) Set(key, url string) bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	_,present:=s.store[key]
	if present {
		return false
	}
	s.store[key]=url
	return true
}

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

func (c *Calc) Add(op1 float64, op2 float64) float64 {
	return op1 + op2
}

func (c *Calc) AddTo(op1 *float64, op2 float64) {
	*op1 = *op1 + op2
}

func (c *Calc) Subtract(op1 float64, op2 float64) float64 {
	return op1 - op2
}

func (c *Calc) Multiply(op1 float64, op2 float64) float64 {
	return op1 * op2
}

func (c *Calc) Divide(op1 float64, op2 float64) (float64, os.Error) {
	if op2 == 0 {
		return 0, os.NewError("Divide " + strconv.Ftoa64(op1, 'f', -1) + " by ZERO!?!")
	}
	return op1 / op2, nil
}

func (c *Calc) Pi() float64 {
	return math.Pi
}

func (c *Calc) Randomize() {
}

func (c *Calc) RandomizeSeed(seed float64) {
}

