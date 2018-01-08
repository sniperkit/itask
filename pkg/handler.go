package xtask

import (
	"reflect"
	"runtime"
)

type Handler struct {
	f    interface{}
	args []interface{}
	ret  []reflect.Value
}

func NewHandler3(f interface{}, args ...interface{}) *Handler {
	res := new(Handler)
	res.f = f
	res.args = args
	return res
}

func (h *Handler) Call() []reflect.Value {
	f := reflect.ValueOf(h.f)
	typ := f.Type()
	if typ.Kind() != reflect.Func {
		panic(errTypeNotFunction)
	}
	// variable parameter, h.args less..
	if typ.NumIn() > len(h.args) {
		panic(errInArgsMissMatch)
	}
	inputs := make([]reflect.Value, len(h.args))
	for i := 0; i < len(h.args); i++ {
		if h.args[i] == nil {
			inputs[i] = reflect.Zero(f.Type().In(i))
		} else {
			inputs[i] = reflect.ValueOf(h.args[i])
		}
	}
	h.ret = f.Call(inputs)
	return h.ret
}

func (h *Handler) BoolCall() bool {
	h.Call()
	if len(h.ret) == 0 {
		panic(errOutCntMissMatch)
	}
	return h.ret[0].Bool()
}

func GetFuncName(h *Handler) string {
	if h == nil || h.f == nil {
		return ""
	}
	return runtime.FuncForPC(reflect.ValueOf(h.f).Pointer()).Name()
}
