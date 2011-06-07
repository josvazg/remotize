package remotize

import (
	"os"
	"reflect"
)

// runtime interface specification implementing ifaceSpec
type rtIfaceSpec struct {
	t reflect.Type
}

func (is *rtIfaceSpec) Name() string {
	if is.t.Kind() == reflect.Interface {
		return is.t.Name()
	}
	if is.t.Kind() == reflect.Ptr {
		return is.t.Elem().Name() + "er"
	}
	return is.t.Name() + "er"
}

func (is *rtIfaceSpec) PkgPath() string {
	if is.t.Kind() == reflect.Ptr {
		return is.t.Elem().PkgPath()
	}
	return is.t.PkgPath()
}

func (is *rtIfaceSpec) NumMethod() int {
	return is.t.NumMethod()
}

func (is *rtIfaceSpec) MethodSpec(i int) methodSpec {
	return &rtMethodSpec{is.t.Method(i), nil}
}

// runtime method specification implemeting methodSpec
type rtMethodSpec struct {
	reflect.Method
	errorType reflect.Type
}

func (m *rtMethodSpec) MethodName() string {
	return m.Name
}

func (m *rtMethodSpec) NumIn() int {
	return m.Type.NumIn()
}

func (m *rtMethodSpec) InName(i int) string {
	return m.Type.In(i).String()
}

func (m *rtMethodSpec) InElem(i int) string {
	return m.Type.In(i).Elem().String()
}

func (m *rtMethodSpec) InPkg(i int) string {
	return m.Type.In(i).PkgPath()
}

func (m *rtMethodSpec) InIsPtr(i int) bool {
	return m.Type.In(i).Kind() == reflect.Ptr
}

func (m *rtMethodSpec) NumOut() int {
	return m.Type.NumOut()
}

func (m *rtMethodSpec) OutName(i int) string {
	return m.Type.Out(i).String()
}

func (m *rtMethodSpec) OutPkg(i int) string {
	return m.Type.Out(i).PkgPath()
}

func (m *rtMethodSpec) OutIsError(i int) bool {
	if m.errorType == nil {
		m.errorType = reflect.TypeOf((*os.Error)(nil)).Elem()
	}
	return m.Type.Out(i) == m.errorType
}

