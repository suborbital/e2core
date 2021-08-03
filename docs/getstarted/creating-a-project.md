# Creating a Project

With subo installed, you can now create a project:

```text
subo create project important-api
```

The project contains two important things: a `Directive.yaml` file, and an example Runnable called `helloworld` written in Rust. The [Directive](https://github.com/suborbital/atmo/tree/32bc83bd9c08ebdc7bce2e8a321dc165f3dc9733/docs/getstarted/concepts/the-directive.md) file defines route handlers and connects [Runnables](https://github.com/suborbital/atmo/tree/32bc83bd9c08ebdc7bce2e8a321dc165f3dc9733/docs/getstarted/concepts/runnables.md) to them.

## Overview

In the Directive file, you'll see a handler set up for you that serves the `POST /hello` route using the `helloworld` Runnable:

{% code title="Directive.yaml" %}
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
{% endcode %}

