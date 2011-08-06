package tool

import (
	"testing"
)

type SomeInterface interface {
}

type SomeStruct struct {
}

type ToolTester interface {
    Integers(int, int8, int16, int32, int16, int64) (int, int8, 
        int16, int32, int16, int64)
    Unsigned(uint, uint8, uint16, uint32, uint16, uint64) (uint, uint8, uint16, 
        uint32, uint16, uint64)
    Floats(float32, float64) (float32, float64)
    Complexs(complex64,complex128) (complex64,complex128)
    Others(bool, string) (bool, string)
    Singlebool(bool) bool
    Aintegers([10]int, [9]int8, [9]int16, [9]int32, [9]int16, 
        [9]int64) ([10]int, [9]int8, [9]int16, [9]int32, [9]int16, [10]int64)
    Aunsigned([10]uint, [9]uint8, [9]uint16, [9]uint32, [9]uint16, 
        [9]uint64) ([10]uint, [9]uint8, [9]uint16, [9]uint32, [9]uint16, 
        [9]uint64)
    Afloats([10]float32, [9]float64) ([9]float32, [10]float64)
    Acomplexs([10]complex64,[9]complex128) ([9]complex64,[10]complex128)
    Types(SomeInterface,SomeStruct) (SomeInterface,SomeStruct)
    Amap(map[string]*[]SomeInterface)
    Pintegers(*int, *int8, *int16, *int32, *int16, *int64) (*int, *int8, 
        *int16, *int32, *int16, *int64)
    Punsigned(*uint, *uint8, *uint16, *uint32, *uint16, *uint64) (*uint, *uint8, 
        *uint16, *uint32, *uint16, *uint64)
    Pfloats(*float32, *float64) (*float32, *float64)
    Pcomplexs(*complex64,*complex128) (*complex64,*complex128)
    Pothers(*bool, *string) (*bool, *string)
    Psinglebool(*bool) *bool
    Ptypes(*SomeInterface,*SomeStruct) (*SomeInterface,*SomeStruct)
    Sintegers([]int, []int8, []int16, []int32, []int16, 
        []int64) ([]int, []int8, []int16, []int32, []int16, []int64)
    Sunsigned([]uint, []uint8, []uint16, []uint32, []uint16, 
        []uint64) ([]uint, []uint8, []uint16, []uint32, []uint16, 
        []uint64)
    Sfloats([]float32, []float64) ([]float32, []float64)
    Scomplexs([]complex64,[]complex128) ([]complex64,[]complex128)
}

func TestTool(t *testing.T) {
    Remotize(new(ToolTester))   
}
