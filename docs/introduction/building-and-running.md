# Building and Running a Project

The `subo` command line tool is used again here to build and run your Atmo project.

## Building

Inside the `important-api` directory run: 

```
subo build .
```

This automatically compiles each of your Runnables in a Docker container and bundles them together in `runnables.wasm.zip` to be used in Atmo. 

```
⏩ START: building runnables in .
ℹ️  🐳 using Docker toolchain
⏩ START: building runnable: helloworld (rust)
    Updating crates.io index
[...]

✅ DONE: bundle was created -> runnables.wasm.zip @ v0.1.0
```

{% hint style="info" %}
If you prefer not to use Docker, you can also [build your Runnables locally](https://github.com/suborbital/subo/blob/main/docs/get-started.md#building-without-docker).
{% endhint %}

## Running a development server

Again, inside the `important-api` directory run: 

```
subo dev
```

This creates a Docker container running the appropriate version of Atmo, copies your `runnables.wasm.zip` into the container, and starts an Atmo server listening on `http://localhost:8080`.

You can test the `/hello` route by sending a POST request to it:

```
curl -X POST http://localhost:8080/hello
```