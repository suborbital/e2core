# Building and Running

The `subo` command line tool is used again here to build and run your Atmo project.

## Building

Inside the `important-api` directory run:

```text
subo build .
```

This automatically compiles each of your Runnables in a Docker container and bundles them together in `runnables.wasm.zip` to be used in Atmo.

```text
⏩ START: building runnables in .
ℹ️  🐳 using Docker toolchain
⏩ START: building runnable: helloworld (rust)
    Updating crates.io index
[...]

✅ DONE: bundle was created -> runnables.wasm.zip @ v0.1.0
```

{% hint style="info" %}
If you prefer not to use Docker, you can also [build your Runnables natively](https://github.com/suborbital/subo/blob/main/docs/get-started.md#building-without-docker).
{% endhint %}

## Running a development server

Now that we have our application bundle built, we can start a development server. In the `important-api` directory, run:

```text
subo dev
```

This creates a Docker container running Atmo, copies your `runnables.wasm.zip` into the container, and starts an Atmo server listening on `http://localhost:8080`.

You can test the `/hello` route in a second terminal by sending a POST request with a body to it:

```text
curl localhost:8080/hello -d 'from the Kármán line!'
```

