# Get started with Atmo 🚀

**NOTE:** These docs are far from complete, but are being actively worked on.

Atmo is a self-hosted that uses a _Runnable bundle_ to run your described application. The bundle includes two things: a Directive, and a set of Runnable WebAssembly modules (functions compiled from various languages such as Rust and Swift).

## Building a bundle
Bundles are built using `subo`, which is the Suborbital CLI tool. You'll need to install `subo` to build a bundle. To install the tool, [visit the subo repository](https://github.com/suborbital/subo).

Once you've installed `subo`, you can use it to build the example project included with this repository. Clone this project, and then run:
```
> subo build ./example-project --bundle
```
The end of this command should read `✅ DONE: bundle was created -> example-project/runnables.wasm.zip`

## Running Atmo
Once you have your runnable bundle, you can run Atmo:
```
> ATMO_HTTP_PORT=8080 make atmo bundle=./example-project/runnables.wasm.zip
```
Atmo will start up and you will begin to see its structured logs in yor terminal. Make a request to `POST localhost:8080/hello` with a request body of `https://github.com`. You will recieve the HTML fetched from `https://github.com/suborbital`.

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
        - modify-url
        - helloworld-rs
      - fn: fetch-test
```
This describes the application being constructed. This declares a route (`POST /hello`) and how to handle that request. The `steps` provided contain a set of instructions on how to handle requests to the `/hello` endpoint. The first step is a `group`, meaning that all of the functions in that group will be executed **concurrently**. The second step is a single function that uses the [Runnable API](https://github.com/suborbital/hive-wasm) to make an HTTP request. The API is continually evolving to include more capabilities. In addition to making HTTP requests, it includes logging abilities and more.

For each function executed, its result gets stored in the request handler's `state`. The `state` is used to pass values between functions, since they are completely isolated and independent from one another. The `modify-url` function takes the request body (in this case, a URL), and modifies it (in this case, adding `/suborbital` to it). The second step (`fetch-test`) takes that modified URL and makes an HTTP request to fetch it. The final function's output is used as the response data for the request.

## Coming soon
Further functionality is incoming along with improved docs, more examples, and an improved Directive format. Visit [the Suborbital website](https://suborbital.dev) to sign up for email updates related to new versions of Atmo.