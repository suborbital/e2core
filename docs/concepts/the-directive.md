# The Directive

The Directive is a declarative file that allows you to describe your application's business logic. By describing your application declaratively, you can avoid all of the boilerplate code that normally comes with building a web service such as binding to ports, setting up TLS, constructing a router, etc.

Here's an example Directive:

```yaml
identifier: com.suborbital.guide
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
          onErr:
            any: continue

      - fn: fetch

  - type: request
    resource: /set/:key
    method: POST
    steps:
      - fn: cache-set

  - type: request
    resource: /get/:key
    method: GET
    steps:
      - fn: cache-get
```

This directive encapsulates all of the logic for your application. It describes three endpoints and the logic needed to handle them. Each handler describes a set of `steps` that composes a series of Runnables to handle the request.

Atmo uses the Directive to build your application and run it automatically, without any need to write boilerplate yourself.

