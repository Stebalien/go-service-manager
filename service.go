package servicemanager

import (
	"context"
	"io"
	"reflect"
)

type service struct {
	service reflect.Value
	deps    []*service
	rdeps   []*service
}

func (s *service) start(ctx context.Context) error {
	if st := s.service.MethodByName("Start"); st != (reflect.Value{}) {
		arguments := make([]reflect.Value, st.Type().NumIn())
		arguments[0] = reflect.ValueOf(ctx)
		for i, dep := range s.deps {
			arguments[i+1] = dep.service
		}
		err := st.Call(arguments)[0]
		if err == (reflect.Value{}) || err.IsNil() {
			return nil
		}
		return err.Interface().(error)
	}
	return nil
}

func (s *service) stop() error {
	if closer, ok := s.service.Interface().(io.Closer); ok {
		return closer.Close()
	}
	return nil
}
