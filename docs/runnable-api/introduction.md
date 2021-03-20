# Introduction

The Runnables that you write for your Atmo application are compiled to WebAssembly, and are run in a controlled sandbox. The **Runnable API** is the set of capabilities Atmo grants to the sandbox which can be used to build your application's logic.

When a Runnable is handling a particular request, Atmo binds that request to the module while it's being run. The Runnable API allows your code to access everything about the request, and also gives you the ability to access the "outside world" by giving functions for HTTP requests, accessing static files, logging, and more. This section describes all of the capabilities available via the Runnable API and how to use them in Rust and Swift Runnable code.

The Runnable API is provided via a library for each of the supported languages, and simply needs to be imported to turn your module into a Runnable. `subo` will configure all of this on your behalf.

The first and most basic part of the Runnable API is the `Runnable` interface \(also known as a Rust trait or Swift protocol\). Every Runnable you write will provide an instance of an object that conforms to this interface. It is very simple, and only requires on method, `run`.

{% hint style="success" %}
The Rust Runnable API crate is considered stable
{% endhint %}

In Rust:

```rust
pub trait Runnable {
    fn run(&self, input: Vec<u8>) -> Result<Vec<u8>, RunErr>;
}
```

{% hint style="warning" %}
The Swift Runnable API library is still considered experimental, and tends to lag slightly behind Rust in terms of available features.
{% endhint %}

And in Swift:

```swift
public protocol Runnable {
    func run(input: String) -> String
}
```

Your Runnable object will be created automatically by `subo` when you use the `create runnable` command. All you need to do is write your logic within the `run` method, and Atmo will handle executing it.

There are several namespaces available in the Runnable API, each are discussed in the following pages.

* [req](request.md)
* [http](http.md)
* [cache](cache.md)
* [file](file.md)
* [log](https://github.com/suborbital/atmo/tree/215d8b0db4673915847a5fd25d4d5c84b8d89186/docs/runnable-api/log.md)

When handling an HTTP request, the input to the `run` method will be a **summary** of the request being handled, not the request itself. The full details of the request are available using the `req` namespace, which will be discussed next.

