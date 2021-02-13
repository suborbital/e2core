package rt

import (
	"sync"

	"golang.org/x/sync/errgroup"
)

// Group represents a group of job results
type Group struct {
	results []*Result
	sync.Mutex
}

// NewGroup creates a new Group
func NewGroup() *Group {
	g := &Group{
		results: []*Result{},
		Mutex:   sync.Mutex{},
	}

	return g
}

// Add adds a job result to the group
func (g *Group) Add(result *Result) {
	g.Lock()
	defer g.Unlock()

	if g.results == nil {
		g.results = []*Result{}
	}

	g.results = append(g.results, result)
}

// Wait waits for all results to come in and returns an error if any arise
func (g *Group) Wait() error {
	g.Lock()
	defer g.Unlock()

	wg := errgroup.Group{}

	for i := range g.results {
		res := g.results[i]

		wg.Go(func() error {
			_, err := res.Then()
			return err
		})
	}

	return wg.Wait()
}
