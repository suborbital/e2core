# Handling requests

When a Runnable is used to handle an HTTP request, Atmo will bind that request to the Runnable. The `req` namespace of the Runnable API can then be used to access all of the information about the request. Note if the Runnable is not being used to handle a request, then all methods in the `req` namespace will return empty or an error.

For Rust, these methods are available under the `req` module, for example `req::method()`. For Swift, they are prefixed with `Req`, for example `Suborbital.ReqMethod()`. For TypeScript/AssemblyScript, they are prefixed with `req`, for example `import { reqState } from '@suborbital/suborbital'`

The following namespace methods are available:

## State

Returns the value from [request state](../usage/managing-state.md) for the provided key

```rust
pub fn state(key: &str) -> Option<String>
```

```typescript
function reqState(key: string): string
```

```swift
public func State(key: String) -> String
```

## Method

Returns the HTTP method for the request

```rust
pub fn method() -> String
```

```typescript
function reqMethod(): string
```

```swift
public func ReqMethod() -> String
```

## URL

Returns the full URL of the request

```rust
pub fn url() -> String
```

```typescript
function reqURL(): string
```

```swift
public func ReqURL() -> String
```

## ID

Returns the unique ID assigned to the request by Atmo

```rust
pub fn id() -> String
```

```typescript
function reqID(): string
```

```swift
public func ReqID() -> String
```

## Body

Returns the full request body as bytes

```rust
pub fn body_raw() -> Vec<u8>
```

```typescript
function reqBody(): ArrayBuffer
```

```swift
public func ReqBodyRaw() -> String
```

## Body Field

Returns the value for the provided key, if the request body is formatted as JSON

```rust
pub fn body_field(key: &str) -> String
```

```typescript
function reqBodyField(key: string): string
```

```swift
public func ReqBodyField(key: String) -> String
```

## Header

Returns the header value for the provided key

```rust
pub fn header(key: &str) -> String
```

```typescript
function reqHeader(key: string): string
```

```swift
public func ReqHeader(key: String) -> String
```

## URL Parameter

Returns the URL parameter for the provided key, for example `/api/v1/user/:uuid`

```rust
pub fn url_param(key: &str) -> String
```

```typescript
function reqURLParam(key: string): string
```

```swift
public func ReqParam(key: String) -> String
```

