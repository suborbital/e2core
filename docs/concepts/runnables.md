# Runnables

When building an application with Atmo, you segment your application's logic into individual functions known as **Runnables**. A Runnable can be written in any of the supported languages \(such as TypeScript, Rust or Swift\), and is compiled to WebAssembly when you build it.

Runnables are completely independent from one another, and have no knowledge of each other's execution. Runnables take an input from Atmo, use the **Runnable API** to run your application logic, and then return an output.

{% hint style="info" %}
You can see some example Runnables in the [example project](https://github.com/suborbital/atmo/tree/main/example-project).
{% endhint %}

Atmo loads a **Bundle** of Runnables at startup and uses your application **Directive** \(discussed next\) to set up and execute your application. Runnables are executed using a job scheduler, meaning that Atmo will "figure out" how to run your application as you've designed, rather than needing to imperatively call functions and structure a large code project like you might be used to with other frameworks.

The **Runnable API** is a library that you include with your application code to gain access to resources such as logging, caching, and access to the network. Atmo dynamically binds resources to your Runnables at runtime, meaning you can swap out various components such as the cache being used without re-writing any code. The CLI tool **subo** takes care of setting up projects, creating Runnables, and building Bundles.

