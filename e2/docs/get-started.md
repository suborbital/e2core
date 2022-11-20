# Get started

Subo includes the WebAssembly toolchain for Suborbital projects.

The Suborbital Development Platform aims for Wasm to be a first-class citizen. `subo` is the toolchain for building Wasm Runnables for [Reactr](https://github.com/suborbital/reactr) and [Atmo](https://github.com/suborbital/atmo). The `subo` CLI can build Wasm Runnables, and can package several Wasm Runnables into a deployable bundle.

Building a Runnable in languages other than Go is designed to be simple and powerful:
```rust
impl runnable::Runnable for Example {
    fn run(&self, input: Vec<u8>) -> Option<Vec<u8>> {
        let in_string = String::from_utf8(input).unwrap();
    
        Some(String::from(format!("hello {}", in_string)).as_bytes().to_vec())
    }
}
```
subo will package your Runnable into a Wasm module that can be used by Reactr or Atmo and run just like any other Runnable! You can see examples of Runnables in the [Reactr repository](https://github.com/suborbital/reactr/tree/main/rwasm/testdata).

## Create a project
To create a new project for Atmo or Reactr, use `subo create project <name>`. This will create a new folder which contains a Directive.yaml and an example Runnable.

Full options for `create project`:
```
create a new project for Atmo or Reactr

Usage:
  subo create project <name> [flags]

Flags:
      --branch string      git branch to download templates from (default "main")
  -h, --help               help for project
      --update-templates   update with the newest templates
```

## Create a Runnable
To create a new Runnable, use the create runnable command:
```
> subo create runnable <name>
```
Rust is chosen by default, but if you prefer Swift, just pass `--lang=swift`! You can now use the Runnable API to build your function. A directory is created for each Runnable, and each contains a `.runnable.yaml` file that includes some metadata.

The full options for `create runnable`:
```
Usage:
  subo create <name> [flags]

Flags:
      --branch string      git branch to download templates from (default "main")
      --dir string         the directory to put the new runnable in (default "/Users/cohix-16/Workspaces/suborbital/subo")
  -h, --help               help for create
      --lang string        the language of the new runnable (default "rust")
      --namespace string   the namespace for the new runnable (default "default")
      --update-templates   update with the newest runnable templates
```

## Building Wasm Runnables
**It is recommended that Docker be installed to build Wasm Runnables. See below if you do not have Docker installed.**
 
To build your Runnable into a Wasm module for Reactr or Atmo, use the build command:
```
> subo build .
```
If the current working directory is a Runnable, subo will build it. If the current directory contains many runnables, subo will build them all. Any directory with a `.runnable.yaml` file is considered a Runnable and will be built. Building Runnables is not fully tested on Windows.

## Bundles
By default, subo will write all of the Runnables in the current directory into a bundle. Atmo uses Runnable bundles to help you build powerful web services by composing Runnables declaratively. If you want to skip bundling, you can pass `--no-bundle` to `subo build`

The resulting bundle can also be used with a Reactr instance by calling `h.HandleBundle({path/to/bundle})`. See the [Reactr Wasm instructions](https://github.com/suborbital/reactr/blob/master/docs/wasm.md) for details.

The full options for `build`:
```
Usage:
  subo build [dir] [flags]

Flags:
      --docker      pass --docker to automatically build a Docker image based on your project's Dockerfile. It will be tagged with the 'identifier' and 'appVersion' from your Directive
  -h, --help        help for build
      --native      if passed, build runnables using native toolchain rather than Docker
      --no-bundle   if passed, a .wasm.zip bundle will not be generated
```

## Building without Docker
If you prefer not to use Docker, you can use the `--native` flag. This will cause subo to use your local machine's toolchain to build Runnables instead of Docker containers. You will need to install the toolchains yourself:
- Rust: Install the latest Rust toolchain and the additional `wasm32-wasi` target.
- Swift: Install the [SwiftWasm](https://book.swiftwasm.org/getting-started/setup.html) toolchain. If using macOS, ensure XCode developer tools are installed (xcrun is required).

`subo` is continually evolving alongside [Reactr](https://github.com/suborbital/reactr) and [Atmo](https://github.com/suborbital/atmo).

## Suborbital Runnable API
Reactr and Atmo provide an [API](https://atmo.suborbital.dev/runnable-api/introduction) which gives Wasm Runnables the ability to access resources and communicate with the host application. This API currently has capabilities such as:
- The ability to make HTTP requests
- Structured logging
- Access to persistent cache
- Access to a static filesystem
- Database access

This API will soon have:
- The ability to render templates
- Access to blob storage
