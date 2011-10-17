package main

import (
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
)

// Some type without interface
type URLStore struct {

}

func (s *URLStore) Get(key string) string {
	fmt.Println("Get(", key, ") (implementation irrelevant)")
	return "something"
}

func (s *URLStore) Set(key, url string) bool {
	fmt.Println("Set(", key, ",", url, ") (implementation irrelevant)")
	return true
}

// Some interface
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
	fmt.Println("Randomized!")
}

func (c *Calc) RandomizeSeed(seed float64) {
	fmt.Println(seed, "randomized!")
}

func (c *Calc) Connect(r io.Reader) io.Writer {
	return nil
}

