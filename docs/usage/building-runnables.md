# Building Runnables

{% hint style="info" %}
This page is far from complete, but is being actively worked on.
{% endhint %}

To build your own Runnables, you can use the Suborbital Rust or Swift Runnable API libraries.

### Create a Runnable

You can create a new Runnable with subo:

```text
> subo create runnable myfunction
```

By default, Rust will be used. To use Swift, pass `--lang`:

```text
> subo create runnable myswiftfunction --lang=swift
```

###  Using the API

Full documentation for the Runnable API is coming soon, but here are the basics:

There are currently four namespaces available:

* http: making HTTP requests
* cache: accessing a connected cache
* log: application logging
* request: access information about the request being handled

For example, in Swift:

{% hint style="warning" %}
The Swift Runnable API library is still considered experimental
{% endhint %}

```swift
import Suborbital

Suborbital.LogInfo(msg: "important information to be logged")
```

```swift
import Suborbital

Suborbital.CacheSet(key: "user-jnirapiu89q", value: "{username: suborbital}", ttl: 0)
```

And in Rust:

{% hint style="success" %}
The Rust Runnable API crate is considered stable
{% endhint %}

```rust
use suborbital::http;

let data = http::get("https://google.com");
```

```rust
use suborbital::cache;

let val = cache::get("user-jnirapiu89q").unwrap_or_default();
```

Please use `subo` to create Runnables, as the above code examples are incomplete snippets.

