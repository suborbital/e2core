package rt

// Ctx is a Job context
type Ctx struct {
	*Capabilities
}

func newCtx(caps *Capabilities) *Ctx {
	c := &Ctx{
		Capabilities: caps,
	}

	return c
}

// Do runs a new job
func (c *Ctx) Do(job Job) *Result {
	if c.doFunc == nil {
		r := newResult(job.UUID())
		r.sendErr(ErrCapabilityNotAvailable)
		return r
	}

	// set the same capabilities as the Job who called Do
	job.caps = c.Capabilities

	return c.doFunc(&job)
}
