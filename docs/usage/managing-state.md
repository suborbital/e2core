# Managing state

Let's take another look at the example Directive:

```yaml
identifier: com.suborbital.test
appVersion: v0.0.1
atmoVersion: v0.0.6

handlers:
  - type: request
    resource: /hello
    method: POST
    steps:
      - group:
        - fn: modify-url
        - fn: helloworld-rs
          as: hello
      - fn: fetch-test
        with:
          url: modify-url
          logme: hello
```

After each `step`, its function results gets stored in the request handler's `state`. The `state` is an ephemeral set of key/value pairs created for each request. State is used to pass values between functions, since they are completely isolated and unaware of one another.

The `modify-url` function for example takes the request body \(in this case, a URL\), and modifies it \(by adding `/suborbital` to it\).

The second step \(`fetch-test`\) takes that modified URL and makes an HTTP request to fetch it.

There are two clauses, `as` and `with` that make working with request state easier.

## As

`as` will assign the value returned by a function to a particular name in state. In the above example, `helloworld-rs` is saved into state as `hello`. You can think of this just like storing a value into a variable!

For example, the request state after the first step will look like this:

```text
{
    "modify-url": "https://github.com/suborbital"
    "hello": "hello github.com"
}
```

## With

`with` allows the developer to pass a "desired state" into a given function. Rather than passing the entire state with the existing keys, the developer can optionally define a custom state by choosing aliases for some or all of the keys available in request state. This is just like choosing parameter values to pass into function arguments!

For example, the `fetch-test` function above will recieve a state object that looks like this:

```text
{
    "url": "https://github.com/suborbital",
    "logme": "hello github.com"
}
```

`subo` will validate your directive to help ensure that your Directive is correct, including validating that you're not accessing keys that don't exist in state.

