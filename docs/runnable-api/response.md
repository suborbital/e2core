# Handling requests

When a Runnable is used to handle an HTTP request, Atmo will bind that request to the Runnable. The `resp` namespace of the Runnable API can then be used to modify the response that Atmo will send to the caller.

For Rust, these methods are available under the `resp` module, for example `resp::set_header()`. For Swift, they are prefixed with `Resp`, for example `Suborbital.RespSetHeader()`

The following namespace methods are available:

## Response header

Sets an HTTP response header

```rust
pub fn set_header(key: &str, val: &str)
```

```swift
// Swift not yet available
```

## Content-Type
An alias of `set_header` that allows easily setting the response Content-Type

```rust
pub fn content_type(ctype: &str)
```

```swift
// Swift not yet available
```