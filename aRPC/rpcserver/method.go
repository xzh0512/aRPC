package rpcserver

import (
	"go/ast"
	"log"
	"reflect"
	"sync/atomic"
)

type methodType struct {
	method    reflect.Method //方法本身，名字啥的都有
	ArgType   reflect.Type
	ReplyType reflect.Type
	numCalls  uint64
}

//实现三个方法，调用次数，创建两个新类型实例

func (m *methodType) NumCalls() uint64 {
	return atomic.LoadUint64(&m.numCalls)
}
func (m *methodType) newArgv() reflect.Value {
	var argv reflect.Value
	if m.ArgType.Kind() == reflect.Ptr {
		argv = reflect.New(m.ArgType.Elem())
	} else {
		argv = reflect.New(m.ArgType).Elem()
	}
	return argv
}

// reply must be a pointer type
func (m *methodType) newReply() reflect.Value {
	replyv := reflect.New(m.ReplyType.Elem())
	//初始化一下元素
	switch m.ReplyType.Elem().Kind() {
	case reflect.Map:
		replyv.Elem().Set(reflect.MakeMap(m.ReplyType.Elem()))
	case reflect.Slice:
		replyv.Elem().Set(reflect.MakeSlice(m.ReplyType.Elem(), 0, 0))
	default:
		break
	}
	return replyv
}

// 为每个结构体定义成一个服务
type service struct {
	name     string
	typ      reflect.Type
	instance reflect.Value //结构体实例
	method   map[string]*methodType
}

func newService(instance interface{}) *service {
	s := new(service)
	s.instance = reflect.ValueOf(instance)
	s.typ = reflect.TypeOf(instance)
	s.name = reflect.Indirect(s.instance).Type().Name()
	if !ast.IsExported(s.name) {
		log.Fatalf("rpc server: %s is not a valid service name", s.name)
	}
	//对方法进行注册
	s.registerServer()
	return s
}
func (s *service) registerServer() {
	s.method = make(map[string]*methodType)
	for i := 0; i < s.typ.NumMethod(); i++ {
		method := s.typ.Method(i)
		mType := method.Type
		if mType.NumIn() != 3 || mType.NumOut() != 1 {
			continue
		}
		if mType.Out(0) != reflect.TypeOf((*error)(nil)).Elem() {
			continue
		}
		argType, replyType := mType.In(1), mType.In(2)
		if !isExportedOrBuiltinType(argType) || !isExportedOrBuiltinType(replyType) {
			continue
		}
		s.method[method.Name] = &methodType{
			method:    method,
			ArgType:   argType,
			ReplyType: replyType,
		}
		log.Printf("rpc server: register %s.%s\n", s.name, method.Name)
	}
}
func isExportedOrBuiltinType(t reflect.Type) bool {
	return ast.IsExported(t.Name()) || t.PkgPath() == ""
}
func (s *service) call(m *methodType, argv, replyv reflect.Value) error {
	atomic.AddUint64(&m.numCalls, 1) //调用次数+1
	f := m.method.Func
	args := []reflect.Value{
		s.instance,
		argv,
		replyv,
	}
	//f.Call(args) 返回的是一个 []reflect.Value，它包含了方法调用的返回值。
	returnValues := f.Call(args)
	//reflect.Value 类型提供了 Interface() 方法，该方法返回 reflect.Value 对应的实际值的接口表示。
	if errInter := returnValues[0].Interface(); errInter != nil {
		return errInter.(error)
	}
	return nil
}
