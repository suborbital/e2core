# Creating handlers

{% hint style="info" %}
If you haven't created a project yet, see [Get started](../getstarted/) first.
{% endhint %}

Your project contains a `Directive.yaml` file that controls your entire application. The Directive is included in the Runnable Bundle used by Atmo to run your application.

The Directive has some metadata such as a unique application identifier and a version number, as well as some handlers.

Each handler tells Atmo how to handle a **resource.** A resource is an input that Atmo makes available via HTTP endpoints, event handlers, and more. To start, Atmo supports handlers for HTTP requests, particulary designed to help building web APIs. Here is an example Directive:

```yaml
identifier: com.suborbital.test
appVersion: v0.0.1
atmoVersion: v0.4.2

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

This describes the application being constructed. It declares a resource \(`HTTP POST /hello`\) and a set of `steps` to handle that request. The `steps` are a set of Runnable functions to be **composed** when handling requests to the `/hello` endpoint.

There are two types of `step`. The first step you see above is a `group`, meaning that all of the functions in that group will be executed **concurrently**.

The second step shown above is a single `fn` , which calls a Runnable that uses the [Runnable API](../runnable-api/introduction.md) to make an HTTP request. The API is continually evolving to include more capabilities. In addition to making HTTP requests, it includes logging, database connections, caching, and more.

The output of the final function in a handler is used as the response data for the request, by default. If you wish to use the output from a different function, you can include the `response` option in your handler, listing the name of the function to use as a response. If the final step is a group, then the `response` clause must be included.

For example: 
```yaml
steps:
  - group:
    - fn: modify-url
    - fn: helloworld-rs
      as: hello
  - fn: fetch-test
    with:
      url: modify-url
      logme: hello
response: hello
```

Your application can contain as many handlers as needed, and functions can be re-used among many handlers. Each Runnable in your project can be called by its name. The `subo` tool will validate your directive to ensure it is not calling any Runnables that don't exist in your project.

The `as` and `with` clauses shown above will be discussed [next](managing-state.md).

