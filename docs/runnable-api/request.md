# Handling requests

When a Runnable is used to handle an HTTP request, Atmo will bind that request to the Runnable. The `req` namespace of the Runnable API can then be used to access all of the information about the request. Note if the Runnable is not being used to handle a request, then all methods in the `req` namespace will return empty or an error.

For Rust, these methods are available under the `req` module, for example `req::method()`. For Swift, they are prefixed with `Req`, for example `Suborbital.ReqMethod()`. For TypeScript/AssemblyScript, they are prefixed with `req`, for example `import { reqState } from '@suborbital/suborbital'`

The following namespace methods are available:

## State

Returns the value from [request state](../usage/managing-state.md) for the provided key:

Rust:

```rust
pub fn state(key: &str) -> Option<String>
```

AssemblyScript:

```typescript
function reqState(key: string): string
```

Swift:

```swift
public func State(key: String) -> String
```

## Method

Returns the HTTP method for the request:

Rust:

```rust
pub fn method() -> String
```

AssemblyScript:

```typescript
function reqMethod(): string
```

Swift:

```swift
public func ReqMethod() -> String
```

## URL

Returns the full URL of the request:

Rust:

```rust
pub fn url() -> String
```

AssemblyScript:

```typescript
function reqURL(): string
```

Swift:

```swift
public func ReqURL() -> String
```

## ID

Returns the unique ID assigned to the request by Atmo:

Rust:

```rust
pub fn id() -> String
```

AssemblyScript:

```typescript
function reqID(): string
```

Swift:

```swift
public func ReqID() -> String
```

## Body

Returns the full request body as bytes:

Rust:

```rust
pub fn body_raw() -> Vec<u8>
```

AssemblyScript:

```typescript
function reqBody(): ArrayBuffer
```

Swift:

```swift
public func ReqBodyRaw() -> String
```

## Body Field

Returns the value for the provided key, if the request body is formatted as JSON:

Rust:

```rust
pub fn body_field(key: &str) -> String
```

AssemblyScript:

```typescript
function reqBodyField(key: string): string
```

Swift:

```swift
public func ReqBodyField(key: String) -> String
```

## Header

Returns the header value for the provided key:

Rust:

```rust
pub fn header(key: &str) -> String
```

AssemblyScript:

```typescript
function reqHeader(key: string): string
```

Swift:

```swift
public func ReqHeader(key: String) -> String
```

## URL Parameter

Returns the URL parameter for the provided key, for example `/api/v1/user/:uuid`

Rust:

```rust
pub fn url_param(key: &str) -> String
```

AssemblyScript:

```typescript
function reqURLParam(key: string): string
```

Swift:

```swift
public func ReqParam(key: String) -> String
```

