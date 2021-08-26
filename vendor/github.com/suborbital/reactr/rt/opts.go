package rt

import "runtime"

// Option is a function that modifies workerOpts
type Option func(workerOpts) workerOpts

//PoolSize returns an Option to set the worker pool size
func PoolSize(size int) Option {
	return func(opts workerOpts) workerOpts {
		opts.poolSize = size
		return opts
	}
}

// Autoscale returns an Option that enables autoscaling and sets the max number of threads
func Autoscale(max int) Option {
	return func(opts workerOpts) workerOpts {
		if max == 0 {
			max = runtime.NumCPU()
		}

		opts.autoscaleMax = max
		return opts
	}
}

//TimeoutSeconds returns an Option with the job timeout seconds set
func TimeoutSeconds(timeout int) Option {
	return func(opts workerOpts) workerOpts {
		opts.jobTimeoutSeconds = timeout
		return opts
	}
}

//RetrySeconds returns an Option to set the worker retry seconds
func RetrySeconds(secs int) Option {
	return func(opts workerOpts) workerOpts {
		opts.retrySecs = secs
		return opts
	}
}

//MaxRetries returns an Option to set the worker maximum retry count
func MaxRetries(count int) Option {
	return func(opts workerOpts) workerOpts {
		opts.numRetries = count
		return opts
	}
}

// PreWarm sets the worker to pre-warm itself to minimize cold start time.
// if not enabled, worker will "warm up" when it receives its first job.
func PreWarm() Option {
	return func(opts workerOpts) workerOpts {
		opts.preWarm = true
		return opts
	}
}
