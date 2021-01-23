# Get started

Atmo is a self-hosted platform that uses a _Runnable bundle_ to run your described application. The bundle includes two things: a [Directive](../concepts/the-directive.md), and a set of [Runnables](../concepts/runnables.md) \(WebAssembly modules compiled from various languages such as Rust and Swift\). A bundle contains everything needed to run your application.

{% hint style="info" %}
**You'll need to install the subo CLI tool and Docker to use Atmo**. 

To install the tool, [visit the subo repository](https://github.com/suborbital/subo).

Docker is used to build Runnables and run the Atmo development server.
{% endhint %}

## Creating a project

You can get started with Atmo by cloning the [repo](https://github.com/suborbital/atmo) which contains an `example-project`, or by using the `subo` CLI.

Once you have subo installed, you can create a project:

```text
> subo create project important-api
```

The project contains two things: a `Directive.yaml` file, and your first Runnable called `helloworld` 

If you open the Directive, you'll see a handler set up for you that serves the `POST /hello` route using the helloworld Runnable.

## Building a bundle

To run an Atmo application, we need to create a Runnable Bundle. A Bundle is a `.wasm.zip` file that includes your Directive, along with all of your Runnables compiled to WebAssembly modules. Bundles are built using `subo`. **Note** that you should pass the root of your Atmo project as the first argument:

```bash
subo build . --bundle
```

The end of this command should read `✅ DONE: bundle was created -> example-project/runnables.wasm.zip`

## Running the Atmo development server

Once you have your Runnable bundle, you can run Atmo:

```text
> subo dev
```

Atmo will start up serving on port 8080, and you will begin to see its structured logs in your terminal. 

If you used the example project from the Atmo repository, make a request to `POST localhost:8080/hello` with a request body of `https://github.com`. You will receive HTML fetched from `https://github.com/suborbital`.

