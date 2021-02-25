# Structured logging

Your Runnable code can log to Atmo's structured output using the logging methods.

For Rust, these methods are available under the `log` module, for example `log::info()`. For Swift, they are prefixed with `Log`, for example `Suborbital.LogInfo()`

The following namespace methods are available:

## Info

Logs the message with the 'info' level

```rust
pub fn info(msg: &str)
```

```swift
public func LogInfo(msg: String)
```

## Warn

Logs the message with the 'warn' level

```rust
pub fn warn(msg: &str)
```

```swift
public func LogWarn(msg: String)
```

## Error

Logs the message with the 'err' level

```rust
pub fn error(msg: &str)
```

```swift
public func LogErr(msg: String)
```

