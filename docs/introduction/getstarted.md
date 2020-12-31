# Get Started

**NOTE:** These docs are far from complete, but are being actively worked on.

Atmo is a self-hosted platform that uses a _Runnable bundle_ to run your described application. The bundle includes two things: a Directive, and a set of Runnables \(WebAssembly modules compiled from various languages such as Rust and Swift\).

## Building a bundle

Bundles are built using `subo`, which is the Suborbital CLI tool. You'll need to install `subo` to build a bundle. To install the tool, [visit the subo repository](https://github.com/suborbital/subo).

Once you've installed `subo`, you can use it to build the example project included with [the Atmo repository](https://github.com/suborbital/atmo). To get started, run:

```bash
git clone git@github.com:suborbital/atmo.git
cd atmo
subo build ./example-project --bundle
```

The end of this command should read `✅ DONE: bundle was created -> example-project/runnables.wasm.zip`

## Running Atmo

Once you have your Runnable bundle, you can run Atmo:

```text
> ATMO_HTTP_PORT=8080 make atmo bundle=./example-project/runnables.wasm.zip
```

Atmo will start up and you will begin to see its structured logs in your terminal. Make a request to `POST localhost:8080/hello` with a request body of `https://github.com`. You will receive HTML fetched from `https://github.com/suborbital`.

## Using Docker

If you prefer using Docker, you can locally build and run Atmo in Docker using:

```text
> make atmo/docker dir=example-project
```

## How it works
If you explore the `example-project` directory, you will see several Runnables (`fetch-test`, `modify-url`, etc.) and a `Directive.yaml` file. Each folder represents an Atmo function, and the Directive is responsible for describing how those functions should be used. The Directive looks like this:
```yaml
identifier: com.suborbital.test
version: v0.0.1

handlers:
  - type: request
    resource: /hello
    method: POST
    steps:
      - group:
        - fn: modify-url
        - fn: helloworld-rs
          as: hello
      - fn: fetch-test
        with:
          - "url: modify-url"
          - "logme: hello"
```
This describes the application being constructed. It declares a route (`POST /hello`) and a set of `steps` to handle that request. The `steps` are a set of functions to be **composed** when handling requests to the `/hello` endpoint. 

The first step is a `group`, meaning that all of the functions in that group will be executed **concurrently**.

The second step is a single function that uses the [Runnable API](https://github.com/suborbital/hive-wasm) to make an HTTP request. The API is continually evolving to include more capabilities. In addition to making HTTP requests, it includes logging abilities and more.

## State
After each step, its function results gets stored in the request handler's `state`. The `state` is an ephemeral set of key/value pairs created for each request. State is used to pass values between functions, since they are completely isolated and unaware of one another. 

The `modify-url` function for example takes the request body (in this case, a URL), and modifies it (by adding `/suborbital` to it). The second step (`fetch-test`) takes that modified URL and makes an HTTP request to fetch it. The final function's output is used as the response data for the request.

There are two clauses, `as` and `with` that make working with request state easier. `as` will assign the value returned by a function to a particular name in state. In the above example, `helloworld-rs` is saved into state as `hello`. You can think of this just like storing a value into a variable!

`with` allows the developer to pass a "desired state" into a given function. Rather than passing the entire state with the existing keys, the developer can optionally define a custom state by choosing aliases for some or all of the keys available in request state. This is just like choosing parameter values to pass into function arguments!

`subo` will validate your directive to help ensure that your Directive is correct, including validating that you're not accessing keys that don't exist in state.

## Coming soon
Further functionality is incoming along with improved docs, more examples, and an improved Directive format. Visit [the Suborbital website](https://suborbital.dev) to sign up for email updates related to new versions of Atmo.