# Building a Bundle

To run your Atmo application, we need to create a Runnable Bundle. A Bundle is a `.wasm.zip` file that includes your Directive, along with all of your Runnables compiled to WebAssembly modules. Bundles are built using `subo`. **Note** that you should pass the root of your Atmo project as the first argument:

```bash
subo build . --bundle
```

The end of this command should read:

`✅ DONE: bundle was created -> ./runnables.wasm.zip`

## Running the Atmo development server

Once you have your Runnable Bundle, you can run Atmo:

```text
> subo dev
```

Atmo will start up serving on port 8080, and you will begin to see its structured logs in your terminal. Make a request to `POST localhost:8080/hello` with a request body to see it in action.

{% hint style="info" %}
The version of Atmo being run by `subo dev` is dictated by the `atmoVersion` key in your Directive.
{% endhint %}

Continue on to learn how to operate Atmo in real-world scenarios.

