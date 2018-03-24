package servicemanager_test

import (
	"context"
	"fmt"
	"testing"

	sm "github.com/Stebalien/go-service-manager"
)

func TestSM(t *testing.T) {
	init := sm.NewInit()

	f := new(foo)
	b := new(bar)
	fs := (FooService)(f)
	bs := (BarService)(b)
	init.Register(&bs)
	init.Register(&fs)

	sm, err := init.Start(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	var foo FooService
	sm.Get(&foo)
	if foo != nil {
		foo.Foo()
	}
	var bar BarService
	sm.Get(&bar)
	if bar != nil {
		bar.Bar()
	}
	if err := sm.Close(); err != nil {
		t.Fatal(err)
	}
}

type FooService interface {
	Foo()
}

type BarService interface {
	Bar()
}

type foo struct{}

func (s *foo) Foo() {
	fmt.Println("foo")
}

func (s *foo) Start(ctx context.Context, bs BarService) error {
	fmt.Println("start foo")
	return nil
}

type bar struct{}

func (s *bar) Bar() {
	fmt.Println("bar")
}

func (s *bar) Start(ctx context.Context) error {
	fmt.Println("start bar")
	return nil
}

func (s *bar) Close() error {
	fmt.Println("shutting down")
	return nil
}
