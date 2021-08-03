# Structured logging

Your Runnable code can log to Atmo's structured output using the logging methods.

For Rust, these methods are available under the `log` module, for example `log::info()`. For Swift, they are prefixed with `Log`, for example `Suborbital.LogInfo()` For TypeScript/AssemblyScript, they are prefixed with `log`, for example `import { logInfo } from '@suborbital/suborbital'`

The following namespace methods are available:

## Info

Logs the message with the 'info' level:

Rust:

```rust
pub fn info(msg: &str)
```

AssemblyScript:

```typescript
function logInfo(msg: string): void
```

Swift:

```swift
public func LogInfo(msg: String)
```

## Warn

Logs the message with the 'warn' level:

Rust:

```rust
pub fn warn(msg: &str)
```

AssemblyScript:

```typescript
function logWarn(msg: string): void
```

Swift:

```swift
public func LogWarn(msg: String)
```

## Error

Logs the message with the 'err' level:

Rust:

```rust
pub fn error(msg: &str)
```

AssemblyScript:

```typescript
function logErr(msg: string): void
```

Swift:

```swift
public func LogErr(msg: String)
```

