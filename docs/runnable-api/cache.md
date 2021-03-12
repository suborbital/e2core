# Accessing cache

Runnables can access an attached cache \(such as Redis\) using the `cache` namespace of the Runnable API. Atmo will configure the cache, and will bind it to the Runnable at runtime. Atmo provides a default in-memory cache if no external cache is connected.

Documentation for connecting said external cache to Atmo is coming soon.

For Rust, these methods are available under the `cache` module, for example `cache::get()`. For Swift, they are prefixed with `Cache`, for example `Suborbital.CacheGet()`

The following namespace methods are available:

## Set

Set a given key's value in the cache. The provided TTL is in seconds.

```rust
pub fn set(key: &str, val: Vec<u8>, ttl: i32)
```

```swift
public func CacheSet(key: String, value: String, ttl: Int)
```

## Get

Get the provided key from the cache.

```rust
pub fn get(key: &str) -> Result<Vec<u8>, RunErr>
```

```swift
public func CacheGet(key: String) -> String
```

Additional cache operations are coming soon.

