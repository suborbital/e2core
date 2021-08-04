# HTTP requests

You can use the `http` namespace of the Runnable API to make HTTP requests from your Runnable code. These methods are currently the only way to access the network from Runnable code. Arbitrary socket and network access is not currently possible.

For Rust, these methods are available under the `http` module, for example `http::get()`. For Swift, they are prefixed with `Http`, for example `Suborbital.HttpGet()` For TypeScript/AssemblyScript, they are prefixed with `http`, for example `import { httpPost } from '@suborbital/suborbital'`

The following namespace methods are available:

## GET

Performs an HTTP GET request

Rust:

```rust
pub fn get(url: &str, headers: Option<BTreeMap<&str, &str>>) -> Result<Vec<u8>, RunErr>
```

AssemblyScript:

```typescript
function httpGet(url: string, headers: Map<string, string> | null): ArrayBuffer
```

Swift:

```swift
public func HttpGet(url: String) -> String
```

## POST

Performs an HTTP POST request:

Rust:

```rust
pub fn post(url: &str, body: Option<Vec<u8>>, headers: Option<BTreeMap<&str, &str>>) -> Result<Vec<u8>, RunErr>
```

AssemblyScript:

```typescript
function httpPost(url: string, body: ArrayBuffer, headers: Map<string, string> | null): ArrayBuffer
```

Swift:

```swift
public func HttpPost(url: String, body: String) -> String
```

## PATCH

Performs an HTTP PATCH request:

Rust:

```rust
pub fn patch(url: &str, body: Option<Vec<u8>>, headers: Option<BTreeMap<&str, &str>>) -> Result<Vec<u8>, RunErr>
```

AssemblyScript:

```typescript
function httpPatch(url: string, body: ArrayBuffer, headers: Map<string, string> | null): ArrayBuffer
```

Swift:

```swift
public func HttpPatch(url: String, body: String) -> String
```

## DELETE

Performs an HTTP DELETE request:

Rust:

```rust
pub fn delete(url: &str, headers: Option<BTreeMap<&str, &str>>) -> Result<Vec<u8>, RunErr>
```

AssemblyScript:

```typescript
function httpDelete(url: string, headers: Map<string, string> | null): ArrayBuffer
```

Swift:

```swift
public func HttpDelete(url: String) -> String
```

Swift does not yet support passing headers to a request.

