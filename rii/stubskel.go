/*



*/
package rii

import (
	"container/mapper"
	"reflect"
)

var stubs mapper.Mapper

type Stubber interface {
	getInterfaceType() *reflect.InterfaceType
}

func init() {
	stubs=mapper.NewMapper(true,true,new(Stubber))
}











