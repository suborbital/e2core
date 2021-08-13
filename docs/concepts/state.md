# State

Since Runnables are completely unaware of one another when being executed, there needs to be a way to pass data between them. Atmo uses a shared object called the **request state** to accomplish this. Request state is a key/value map that is updated automatically after each step in a handler.

Let's take a look at a handler from the Directive:

```yaml
handlers:
  - type: request
    resource: /hello
    method: POST
    steps:
      - fn: verify-request
      - group:
        - fn: modify-url
        - fn: cache-get
      - fn: fetch
```

Here we can see a request with three steps. The first and third are single functions being called, and the second is a function group.

After each step in the handler, the request state is updated from the output of the functions in that step.

For example, after the **first step**, the state will look like this:

```text
{
    "verify-request": "ok"
}
```

And then after the **second step**:

```text
{
    "verify-request": "ok"
    "modify-url": "https://github.com/suborbital"
    "cache-get": {"Auth-Header": "nuw45tpjno998w3un10nfwe8h"}
}
```

When each step executes, the current request state is made available to the Runnable using **Runnable API** functions.

{% hint style="info" %}
Request state is updated after each **step**, so it is important to note that multiple functions in a **group** will all receive the same state from the beginning of the step, and all of their outputs will be added to state after they've all completed executing.
{% endhint %}

You can access request state like the following Runnable example written in Rust.

```rust
use suborbital::req;

[...]

let url = req::state("modify-url");
```

There are several clauses that allow you to control how the request state is set up \(for example, choosing the key that a function's output is stored in\), which will be covered later.

