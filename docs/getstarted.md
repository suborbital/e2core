# Get started

Atmo is a self-hosted platform that uses a _Runnable bundle_ to run your described application. The bundle includes two things: a [Directive](concepts/the-directive.md), and a set of [Runnables](concepts/runnables.md) \(WebAssembly modules compiled from various languages such as Rust and Swift\). A bundle contains everything needed to run your application.

{% hint style="info" %}
**You'll need to install the subo CLI tool and Docker to use Atmo**.

To install the tool, [visit the subo repository](https://github.com/suborbital/subo).

Docker is used to build Runnables and run the Atmo development server.
{% endhint %}

## Creating a project

Once you have subo installed, you can create a project:

```text
> subo create project important-api
```

The project contains two things: a `Directive.yaml` file, and your first Runnable called `helloworld`

If you open the Directive, you'll see a handler set up for you that serves the `POST /hello` route using the helloworld Runnable.

Read on to learn about the different aspects of an Atmo project, or skip right to [building your application bundle.](usage/building-a-bundle.md)

