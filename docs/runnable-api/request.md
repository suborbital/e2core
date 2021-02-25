# Handling requests

When a Runnable is used to handle an HTTP request, Atmo will bind that request to the Runnable. The `req` namespace of the Runnable API can then be used to access all of the information about the request. Note if the Runnable is not being used to handle a request, then all methods in the `req` namespace will return empty or an error.

For Rust, these methods are available under the `req` module, for example `req::method()`. For Swift, they are prefixed with `Req`, for example `Suborbital.ReqMethod()`

The following namespace methods are available:

## Method

Returns the HTTP method for the request

```rust
pub fn method() -> String
```

```swift
public func ReqMethod() -> String
```

## URL

Returns the full URL of the request

```rust
pub fn url() -> String
```

```swift
public func ReqURL() -> String
```

## ID

Returns the unique ID assigned to the request by Atmo

```rust
pub fn id() -> String
```

```swift
public func ReqID() -> String
```

## Body

Returns the full request body as bytes

```rust
pub fn body_raw() -> Vec<u8>
```

```swift
public func ReqBodyRaw() -> String
```

## Body Field

Returns the value for the provided key, if the request body is formatted as JSON

```rust
pub fn body_field(key: &str) -> String
```

```swift
public func ReqBodyField(key: String) -> String
```

## Header

Returns the header value for the provided key

```rust
pub fn header(key: &str) -> String
```

```swift
public func ReqHeader(key: String) -> String
```

## URL Parameter

Returns the URL parameter for the provided key, for example `/api/v1/user/:uuid`

```rust
pub fn url_param(key: &str) -> String
```

```swift
public func ReqParam(key: String) -> String
```

## State

Returns the value from [request state](../usage/managing-state.md) for the provided key

```rust
pub fn state(key: &str) -> String
```

```swift
public func State(key: String) -> String
```

