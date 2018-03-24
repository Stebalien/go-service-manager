package servicemanager

import (
	"context"
	"fmt"
	"reflect"
	"sync"
)

var contextT = reflect.TypeOf((*context.Context)(nil)).Elem()
var errorT = reflect.TypeOf((*error)(nil)).Elem()

// Init starts a set of services.
//
// This type is thread-safe.
type Init struct {
	implementations map[reflect.Type]reflect.Value
	dependencies    map[reflect.Value][]reflect.Type // maps services to deps

	mu sync.Mutex
}

// NewInit constructs a new service initialize
func NewInit() *Init {
	return &Init{
		implementations: make(map[reflect.Type]reflect.Value),
		dependencies:    make(map[reflect.Value][]reflect.Type),
	}
}

func (init *Init) graph() ([]*service, error) {
	serviceMap := make(map[reflect.Value]*service, len(init.dependencies))
	for v, deps := range init.dependencies {
		s, ok := serviceMap[v]
		if !ok {
			s = &service{
				service: v,
			}
			serviceMap[v] = s
		}

		for _, depT := range deps {
			dep, ok := init.implementations[depT]
			if !ok {
				return nil, fmt.Errorf("service %s is missing dependency %s", v.Type(), depT)
			}
			sd, ok := serviceMap[dep]
			if !ok {
				sd = &service{
					service: v,
				}
				serviceMap[dep] = sd
			}
			s.deps = append(s.deps, sd)
			sd.rdeps = append(sd.rdeps, s)
		}
	}

	services := make([]*service, 0, len(serviceMap))
	for _, service := range serviceMap {
		services = append(services, service)
	}
	return services, nil
}

func (init *Init) finish() (*Manager, error) {
	init.mu.Lock()
	defer init.mu.Unlock()

	services, err := init.graph()
	if err != nil {
		init.mu.Unlock()
		return nil, err
	}

	manager := &Manager{
		running:         services,
		implementations: init.implementations,
	}

	init.dependencies = nil
	init.implementations = nil

	return manager, nil
}

// Start starts all the registered services, in parallel and order of dependency.
//
// Services registered after starting will be started immediately.
func (init *Init) Start(ctx context.Context) (*Manager, error) {
	manager, err := init.finish()
	if err != nil {
		return nil, err
	}
	err = manager.start(ctx)
	if err != nil {
		return nil, err
	}
	return manager, nil
}

// Register registers new services, overriding any existing services registered.
//
// It panics if the service implementation is invalid. See TryRegister
func (init *Init) Register(iface interface{}) {
	if err := init.TryRegister(iface); err != nil {
		panic(err)
	}
}

// TryRegister registers new a new service.
//
// If a Start method is present, the first parameter must be a context and the
// rest must be interfaces of registered services. Start must return an error.
//
// If a Close method is present, it must satisfy the io.Closer interface.
func (init *Init) TryRegister(iface interface{}) error {
	init.mu.Lock()
	defer init.mu.Unlock()

	ty := reflect.TypeOf(iface)
	if ty.Kind() != reflect.Ptr {
		return fmt.Errorf("given interfaces must be pointers to interfaces")
	}
	ty = ty.Elem()
	if ty.Kind() != reflect.Interface {
		return fmt.Errorf("given interfaces must be pointers to interfaces")
	}

	v := reflect.ValueOf(iface).Elem().Elem()
	if vc, ok := init.implementations[ty]; ok {
		if vc == v {
			return nil
		}
		return fmt.Errorf("duplicate services %s and %s for interface %s", v.Type(), vc.Type(), ty)
	}

	if deps, ok := init.dependencies[v]; !ok {
		if start := v.MethodByName("Start"); start != (reflect.Value{}) {
			st := start.Type()
			if st.NumOut() != 1 || st.Out(0) != errorT {
				return fmt.Errorf("expected Start to return an error")
			}
			if st.NumIn() == 0 || st.In(0) != contextT {
				return fmt.Errorf("expected first argument of Start to be a context")
			}
			for i := 1; i < st.NumIn(); i++ {
				argT := st.In(i)
				if argT.Kind() != reflect.Interface {
					return fmt.Errorf("service dependency not an interface: %s", argT)
				}
				deps = append(deps, argT)
			}
		}
		if close := v.MethodByName("Close"); close != (reflect.Value{}) {
			if _, ok := close.Interface().(func() error); !ok {
				return fmt.Errorf("expected Close method to have the signature 'func() error', has signature '%s'", close.Type())
			}
		}
		init.dependencies[v] = deps
	}
	init.implementations[ty] = v
	return nil
}
