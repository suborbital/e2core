# Rust Hive Runnable

To create a Rust-based Wasm Runnable, use the `subo` CLI to build it. Create a directory with the name of your runnable, and add three files: `.hive.yml`, Cargo.toml, and `run.rs`. inside it. The YAML file can be empty, it is just a placeholder for now. Each runnable should look like this:
```
| name-of-runnable
| - .hive.yml
| - Cargo.toml
| - run.rs
```
`name-of-runnable` should match the `name` field in Cargo.toml.

Your `run.rs` should have a `run` function with this signature: 
```rust
#[no_mangle]
pub fn run(input: Vec<u8>) -> Option<Vec<u8>>
```
You can put whatever you want into this function, so long as it'll run in a WASI environment!

Once your runnable is ready, run `subo build` in the parent directory, and every directory with a `.hive.yml` will be built into a WASM runnable, with the resulting file being put inside the runnable directory.

Head back to [the Hive WASM docs](https://github.com/suborbital/hive/blob/master/docs/wasm.md) to learn how to use them!
