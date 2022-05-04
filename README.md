# Velocity

Velocity is an application framework that brings multi-language functions and workflows to any application using WebAssembly. By running ephemeral, stateless, sandboxed functions, Velocity adds infinitely scalable compute to any architecture.

Velocity can start as small as adding a single function to an existing application, grow to be the basis for entire complex systems, and feel integrated into your existing developer tooling the entire way. Velocity can create, build, and deploy functions written in popular languages like JavaScript, TypeScript, Go, Rust, and more.

## Partners
Velocity is designed to be used with **partner applications**. By binding Velocity server to _any existing server_, your application can begin executing functions and workflows using a simple library or HTTP call. Velocity can automatically set up your application to run functions with **zero config**, for example:
* Add a Golang function to a Next.js application
* Add a Rust function to a PHP application
* Add sandboxing to a critical JavaScript function
* Add a JAMStack backend (in any language) to a vanilla React application

## Standalone
When your application grows beyond adding simple functions to a partner, Velocity can act as a standalone backend development framework. Using a declarative description of your application called the _Directive_, Velocity can run individual functions, workflows of chained functions, and more. Velocity can be deployed as a (micro) service within your existing infrastructure, or it can be the framework that brand new applications are built on from the start.

To get started, just install Velocity:
```bash
brew tap suborbital/velocity
brew install velocity
```
(or [download a release]() for other platforms)

And then check out the getting started guides for [React](), [Next.js](), [Deno](), [Lavarel](), [Ruby on Rails](), [Svelte](), [Astro](), [Kubernetes sidecar](), or the [standalone guide]().

Velocity has a detailed [spec](./spec) that details its internal workings, APIs, and includes a reference for creating custom backends. The default backend for Velocity is [Sat]().

Copyright Suborbital Contributors. Apache 2.0 License.