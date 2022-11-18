package scheduler

import (
	"testing"

	"github.com/pkg/errors"
)

func TestReactrJobGroup(t *testing.T) {
	r := New()

	doMath := r.Register("math", math{})

	grp := NewGroup()
	grp.Add(doMath(input{5, 6}))
	grp.Add(doMath(input{7, 8}))
	grp.Add(doMath(input{9, 10}))

	if err := grp.Wait(); err != nil {
		t.Error(errors.Wrap(err, "failed to grp.Wait"))
	}
}

func TestLargeGroup(t *testing.T) {
	r := New()

	doMath := r.Register("math", math{})

	grp := NewGroup()
	for i := 0; i < 50000; i++ {
		grp.Add(doMath(input{5, 6}))
	}

	if err := grp.Wait(); err != nil {
		t.Error(err)
	}
}

func TestLargeGroupWithPool(t *testing.T) {
	r := New()

	doMath := r.Register("math", math{}, PoolSize(3))

	grp := NewGroup()
	for i := 0; i < 50000; i++ {
		grp.Add(doMath(input{5, i}))
	}

	if err := grp.Wait(); err != nil {
		t.Error(err)
	}
}

type groupWork struct{}

// Run runs a groupWork job
func (g groupWork) Run(job Job, ctx *Ctx) (interface{}, error) {
	grp := NewGroup()

	grp.Add(ctx.Do(NewJob("generic", "first")))
	grp.Add(ctx.Do(NewJob("generic", "group work")))
	grp.Add(ctx.Do(NewJob("generic", "group work")))
	grp.Add(ctx.Do(NewJob("generic", "group work")))
	grp.Add(ctx.Do(NewJob("generic", "group work")))

	return grp, nil
}

func (g groupWork) OnChange(change ChangeEvent) error {
	return nil
}

func TestReactrChainedGroup(t *testing.T) {
	r := New()

	r.Register("generic", generic{})
	doGrp := r.Register("group", groupWork{})

	if _, err := doGrp(nil).Then(); err != nil {
		t.Error(errors.Wrap(err, "failed to doGrp"))
	}
}
