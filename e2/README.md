# e2, the Suborbital Extension Engine CLI

e2 is the command-line helper for working with the Suborbital Extension Engine. e2 is used to create and build Wasm plugins, generate new projects and config files, and more over time.

**You do not need to install language-specific tools to get started with WebAssembly and e2!** A Docker toolchain is supported (see below) that can build your plugins without needing to install language toolchains.

## Installing
### macOS (Homebrew)
If you're on Mac (M1 or Intel), the easiest way to install is via `brew`:
```
brew tap suborbital/e2
brew install e2
```

### Install from source (requires Go)
If you use Linux or otherwise prefer to build from source, simply clone this repository or download a [source code release](https://github.com/suborbital/e2core/releases/latest) archive and run:
```
make e2
```
This will install `e2` into your GOPATH (`$HOME/go/bin/e2` by default) which you may need to add to your shell's `$PATH` variable.

e2 does not have official support for Windows.

## Verify installation
Verify e2 was installed:
```
e2 --help
```


## Getting started
**To get started with e2, visit the [Get started guide](./docs/get-started.md).**

## Builders
This repo contains builders for the various languages supported by Wasm Runnables. A builder is a Docker image that can build Runnables into Wasm modules, and is used internally by `subo` to build your code! See the [builders](./builder/docker) directory for more.

## Platforms
The `subo` tool supports the following platforms and operating systems:
|  | x86_64 | arm64
| --- | --- | --- |
| macOS | ✅ | ✅ |
| Linux | ✅ | ✅ |
| Windows* | — | — |

_*On Windows you can use WSL._
 
The language toolchains used by `subo` support the following platforms:
| | x86_64 | arm64 | Docker |
| --- | --- | --- | --- |
| Rust | ✅ | ✅ | ✅ |
| JavaScript | ✅ | ✅ | ✅ |
| TypeScript | ✅ | ✅ | ✅ |
| TinyGo | ✅ | ✅ | ✅ |
| Grain | ✅ | ✅ | ✅ |
| AssemblyScript | ✅ | ✅ | ✅ |
| Swift | ✅ | — | 🟡 &nbsp;(no arm64) |

## Contributing

Please read the [contributing guide](./CONTRIBUTING.md) to learn about how you can contribute to e2! We welcome all types of contribution.

By the way, e2 is also the name of our mascot, and it's pronounced SOO-bo.

![SOS-Space_Panda-Dark-small](https://user-images.githubusercontent.com/5942370/129103528-8b013445-a8a2-44bb-8b39-65d912a66767.png)

Copyright © 2021-2022 Suborbital and contributors.
