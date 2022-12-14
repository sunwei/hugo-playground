package para

import (
	"context"
	"golang.org/x/sync/errgroup"
)

// Workers configures a task executor with the most number of tasks to be executed in parallel.
type Workers struct {
	sem chan struct{}
}

// New creates a new Workers with the given number of workers.
func New(numWorkers int) *Workers {
	return &Workers{
		sem: make(chan struct{}, numWorkers),
	}
}

// Start starts a new Runner.
func (w *Workers) Start(ctx context.Context) (Runner, context.Context) {
	g, ctx := errgroup.WithContext(ctx)
	return &errGroupRunner{
		Group: g,
		ctx:   ctx,
		w:     w,
	}, ctx
}

// Runner wraps the lifecycle methods of a new task set.
//
// Run wil block until a worker is available or the context is cancelled,
// and then run the given func in a new goroutine.
// Wait will wait for all the running goroutines to finish.
type Runner interface {
	Run(func() error)
	Wait() error
}

type errGroupRunner struct {
	*errgroup.Group
	w   *Workers
	ctx context.Context
}

func (g *errGroupRunner) Run(fn func() error) {
	select {
	case g.w.sem <- struct{}{}:
	case <-g.ctx.Done():
		return
	}

	g.Go(func() error {
		err := fn()
		<-g.w.sem
		return err
	})
}
