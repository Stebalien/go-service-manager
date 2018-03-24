package servicemanager

import (
	"context"
	"reflect"
	"sync"
)

// Manager manages a running set of services.
//
// This type is thread-safe.
type Manager struct {
	// Immutable!
	implementations map[reflect.Type]reflect.Value

	stopOnce sync.Once
	stopErr  error
	running  []*service
}

func (sm *Manager) start(ctx context.Context) error {
	return start(ctx, sm.running)
}

// Close stops all running services.
func (sm *Manager) Close() error {
	sm.stopOnce.Do(func() {
		running := sm.running
		sm.running = nil
		sm.stopErr = stop(running)
	})
	return sm.stopErr
}

// Get gets a service given a pointer to an interface.
//
// Panics if the given interface isn't a pointer to an interface.
func (sm *Manager) Get(s interface{}) {
	ty := reflect.TypeOf(s)
	if ty.Kind() != reflect.Ptr {
		panic("can only assign services to pointers")
	}
	ty = ty.Elem()
	if ty.Kind() != reflect.Interface {
		panic("services are interfaces")
	}
	if imp, ok := sm.implementations[ty]; ok {
		reflect.ValueOf(s).Elem().Set(imp)
	}
}
