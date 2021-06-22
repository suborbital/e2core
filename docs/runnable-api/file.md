# Static files

Files in the `static` directory of an Atmo project will be copied into the Runnable Bundle by `subo`. Those files can then be accessed by Runnables. The directory is mounted as a sandboxed read-only filesystem.

For Rust, these methods are available under the `file` module, for example `file::get_static()`.

The following namespace methods are available:

## Get Static

Retrieves the contents of the static file with the given name

```rust
pub fn get_static(name: &str) -> Result<Vec<u8>, RunErr>
```

```typescript
// not yet supported
```

```swift
public func GetStaticFile(name: String) -> String
```

