# VLog: Simple and Safe logging package

`vlog` is the logging package for the Suborbital Development Platform. It is designed to have a minimal performance impact and promote logging safety.

## The default instance
For most users, using `vlog.Default()` is enough. This creates a `Logger` that logs to stdout, uses the `info` log level, and redacts non-string inputs. If you want to gain finer control over the logger, read on!

## Using the logger
The logger uses a simple API to get out of your way:
```golang
// ErrorString logs the input as an error
func (v *Logger) ErrorString(msgs ...interface{}) {}

// Error logs an error object
func (v *Logger) Error(err error) {}

// Warn logs the input as an warning
func (v *Logger) Warn(msgs ...interface{}) {}

// Info logs the input as an info message
func (v *Logger) Info(msgs ...interface{}) {}

// Debug logs the input as debug output
func (v *Logger) Debug(msgs ...interface{}) {}

// Trace logs a function name and returns a function to be deferred, logging the completion of a function
func (v *Logger) Trace(fnName string) func() {}
```
Each method takes in a list of `interface{}` which are appended when logging. For example:
```golang
log.Info("user", user.Email, "completed signin")
```
Will print `(I) user info@example.com completed signup`. The `(I)` indicates the log level (info). How the logger processes the passed in objects is determined by the producer, which is discussed below.

## Log levels
The logger will automatically filter out anything higher than the configured level. For example, if the logger is configured for `LogLevelError`, then the higher levels such as Info, Debug, and Trace will not be logged. The available log level are as follows:
```golang
// LogLevelTrace and others represent log levels
const (
	LogLevelTrace = "trace" // 5
	LogLevelDebug = "debug" // 4
	LogLevelInfo  = "info"  // 3
	LogLevelWarn  = "warn"  // 2
	LogLevelError = "error" // 1
)
```

### The trace level
The `Trace` log method is special, in that it returns a function. This allows for easy function tracing:
```golang
func SomethingAwesome() {
	defer log.Trace("SomethingAwesome")
}
```
This will print something like:
```
(T) SomethingAwesome
[...]
(T) SomethingAwesome completed
```

## Logger options
The default constructor and `vlog.New()` both take a set of `OptionModifier` parameters, which are functions that set the various available options. For example:
```golang
log := vlog.Default(
		vlog.Level(vlog.LogLevelTrace)
	)
```
Passing in options will allow you to tweak the behaviour of the logger. The available options are:
```golang
// Level sets the logging level to one of error, warn, info, debug, or trace (VLOG_LOG_LEVEL env var)
func Level(level string)

// ToFile sets the logger to open the file specified and write logs to it (VLOG_LOG_FILE env var)
func ToFile(filepath string)

// LogPrefix sets a prefix on all of the log messages (VLOG_LOG_PREFIX env var)
func LogPrefix(prefix string)

// EnvPrefix sets the prefix to be used for environment variable settings (replaces VLOG with prefix in env var keys above)
func EnvPrefix(prefix string)

// AppMeta sets the meta object to be included with structured logs (not configurable from env vars)
func AppMeta(meta interface{})

// PreLogHook sets a function that will be called every time something
// is logged. The value will be the structured JSON for the log line
// LogHookFunc has the signature `func([]byte)`
func PreLogHook(hook LogHookFunc)
```
> Note if `ToFile` is used, structured logs are written to the file and plain text logs are duplicated to stdout.

## The Producer
`vlog` uses an object called the `Producer` to process all log lines. `Producer` is an interface type, and its implementation is responsible for taking the input passed into each log method and converting it into a string for logging. The `Producer` that ships with `vlog` is called `defaultProducer`; it logs all strings, but redacts all other types it is given for safety. If logging of structs or other types is needed, it is reccomended that a custom `Producer` is created. Simply copy `defaultproducer.go`, add your own functionality, and pass it in to `vlog.New(producer, opts...)` to create your logger.

## Structured logging
Structured logs are core to vlog, and there are a number of features that make it useful. Things like the log level and timestamp are included by default, and `AppMeta` and `Scope` are two ways to make structured logs even more useful.

An example of a structured log is as follows:
```json
{"log_message":"(I) serving on :443","timestamp":"2020-10-12T20:55:00.644217-04:00","level":3,"app":{"version":"v0.1.1"}}
```

### AppMeta
`AppMeta` (configured using the `Meta()` OptionModifier when instantiating the logger) represents metadata about the running application. The configured meta will be included with every log message. This can be used to indicate the version of the currently running application, for example. The `AppMeta` is included in structured logs under the `app` JSON key. If the object set as AppMeta cannot be JSON marshalled, an error will occur.

### Scope
A `Logger` instance can create a "scoped" instance of itself, which is essentially a clone with a specific scope object attached. Scope can be useful to add a specific request ID to logs related to it, for instance. Calling `logger.CreateScoped(scope interface{})` on a `Logger` will return a new `Logger` that includes the provided object under the `scope` JSON key. If the object set as Scope cannot be JSON marshalled, an error will occur.

A shortcut for setting scope on the logger with `vk` is the `ctx.UseScope()` method on the `vk.Ctx` type. This will automatically create a scoped logger, set it as the logger for that request, and make the scope object available for later use via the `ctx.Scope()` method. 