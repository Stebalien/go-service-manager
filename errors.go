package servicemanager

import (
	"fmt"
	"strings"
)

// ErrCycle is returned when there is a service dependency cycle.
type ErrCycle map[interface{}][]interface{}

func (e ErrCycle) Error() string {
	if len(e) == 0 {
		panic("bad cycle error")
	}
	var builder strings.Builder
	fmt.Fprint(&builder, "dependency cycle: ")
	for p, c := range e {
		fmt.Fprintf(&builder, "%T -> {", p)
		for _, c := range c[:len(c)-1] {
			fmt.Fprintf(&builder, "%T, ", c)
		}
		fmt.Fprintf(&builder, "%T}; ", c[len(c)-1])
	}
	s := builder.String()
	return s[:len(s)-2]
}

type ErrManage struct {
	Start, Stop error
}

func (err *ErrManage) Error() string {
	var builder strings.Builder
	if err.Start != nil {
		fmt.Fprintf(&builder, "errors when starting: %s\n", err.Start)
	}
	if err.Stop != nil {
		fmt.Fprintf(&builder, "errors when stopping: %s\n", err.Stop)
	}
	return builder.String()
}

// ErrService is returned when one or more services return an error.
type ErrService map[interface{}]error

func (errs ErrService) Error() string {
	if len(errs) == 0 {
		panic("bad service error")
	}
	var builder strings.Builder
	for s, err := range errs {
		fmt.Fprintf(&builder, "%T: %s; ", s, err)
	}
	s := builder.String()
	return s[:len(s)-2]
}
