# Modifying responses

When a Runnable is used to handle an HTTP request, Atmo will bind that request to the Runnable. The `resp` namespace of the Runnable API can then be used to modify the response that Atmo will send to the caller.

For Rust, these methods are available under the `resp` module, for example `resp::set_header()`. Swift and TypeScript/AssemblyScript support is coming soon.

The following namespace methods are available:

## Response header

Sets an HTTP response header:

Rust:

```rust
pub fn set_header(key: &str, val: &str)
```

AssemblyScript:

```typescript
// not yet available
```

Swift:

```swift
// not yet available
```

## Content-Type

An alias of `set_header` that allows easily setting the response Content-Type

Rust:

```rust
pub fn content_type(ctype: &str)
```

AssemblyScript:

```typescript
// not yet available
```

Swift:

```swift
// not yet available
```

