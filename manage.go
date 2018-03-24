package servicemanager

import (
	"context"
)

type status struct {
	service *service
	err     error
}

func startGraph(services []*service) map[*service]int {
	graph := make(map[*service]int, len(services))
	for _, service := range services {
		graph[service] = len(service.deps)
	}
	return graph
}

func stopGraph(services []*service) map[*service]int {
	graph := make(map[*service]int, len(services))
	for _, service := range services {
		graph[service] = len(service.rdeps)
	}
	return graph
}

func check(services []*service) error {
	graph := startGraph(services)

	var ready []*service
	for s, pending := range graph {
		if pending == 0 {
			ready = append(ready, s)
		}
	}

	var started = 0
	for len(ready) > 0 {
		started++
		next := ready[len(ready)-1]
		ready = ready[:len(ready)-1]
		for _, rd := range next.rdeps {
			waiting := graph[rd] - 1
			graph[rd] = waiting
			if waiting == 0 {
				ready = append(ready, rd)
			}
		}
	}
	if started < len(services) {
		blocked := make(ErrCycle, len(services)-started)
		for s, waiting := range graph {
			if waiting == 0 {
				continue
			}
			var blocking []interface{}
			for _, d := range s.deps {
				if graph[d] == 0 {
					continue
				}
				blocking = append(blocking, d.service.Interface())
			}
			blocked[s.service.Interface()] = blocking
		}

		return blocked
	}
	return nil
}

func stop(services []*service) error {
	graph := stopGraph(services)
	done := make(chan status)
	stopping := 0
	for s, deps := range graph {
		if deps > 0 {
			continue
		}
		stopping++
		go func(s *service) {
			done <- status{
				service: s,
				err:     s.stop(),
			}
		}(s)
	}

	errs := make(ErrService)
	for stopping > 0 {
		result := <-done
		stopping--
		if result.err != nil {
			errs[result.service.service.Interface()] = result.err
		}

		for _, rd := range result.service.rdeps {
			waiting := graph[rd] - 1
			graph[rd] = waiting
			if waiting == 0 {
				stopping++
				go func(s *service) {
					done <- status{
						service: s,
						err:     s.stop(),
					}
				}(rd)
			}
		}
	}
	if len(errs) > 0 {
		return &ErrManage{Stop: errs}
	}
	return nil
}

func start(ctx context.Context, services []*service) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	if err := check(services); err != nil {
		return err
	}
	graph := startGraph(services)
	done := make(chan status)
	starting := 0
	for s, deps := range graph {
		if deps > 0 {
			continue
		}
		starting++
		go func(s *service) {
			done <- status{
				service: s,
				err:     s.start(ctx),
			}
		}(s)
	}

	var running []*service
	for starting > 0 {
		result := <-done
		starting--
		if result.err != nil {
			var err ErrManage
			var startErrs ErrService
			if ctxErr := ctx.Err(); ctxErr != nil {
				err.Start = ctxErr
			} else {
				startErrs = make(ErrService)
				err.Start = startErrs
				cancel()
			}
			for starting > 0 {
				r := <-done
				starting--
				if r.err != nil {
					running = append(running, r.service)
				} else if startErrs != nil {
					startErrs[r.service.service.Interface()] = r.err
				}
			}
			err.Stop = stop(running)
			return &err
		}

		running = append(running, result.service)
		for _, rd := range result.service.rdeps {
			waiting := graph[rd] - 1
			graph[rd] = waiting
			if waiting == 0 {
				starting++
				go func(s *service) {
					done <- status{
						service: s,
						err:     s.start(ctx),
					}
				}(rd)
			}
		}
	}
	return nil
}
