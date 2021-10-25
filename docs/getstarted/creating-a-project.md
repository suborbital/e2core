# Creating a Project

With subo installed, you can now create a project:

```text
subo create project important-api
```

The project contains two important things: a `Directive.yaml` file, and an example Runnable called `helloworld` written in Rust. The [Directive](../concepts/the-directive.md) file defines route handlers and connects [Runnables](../concepts/runnables.md) to them.

### Overview

In the Directive file, you'll see a handler set up for you that serves the `POST /hello` route using the `helloworld` Runnable:

```yaml
# the Directive is a complete description of your application, including all of its business logic.
# appVersion should be updated for each new deployment of your app.
# atmoVersion declares which version of Atmo is used for the `subo dev` command.

identifier: com.suborbital.important-api
appVersion: v0.1.0
atmoVersion: v0.2.3


handlers:
  - type: request
    resource: /hello
    method: POST
    steps:
      - fn: helloworld
```

