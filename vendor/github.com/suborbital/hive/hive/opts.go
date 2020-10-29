package hive

// Option is a function that modifies workerOpts
type Option func(workerOpts) workerOpts

//PoolSize returns an Option to set the worker pool size
func PoolSize(size int) Option {
	return func(opts workerOpts) workerOpts {
		opts.poolSize = size
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
