# Static Directory

An Atmo project can optionally contain a `static` directory. When present, the `subo` CLI will package the static directory into your application Bundle. Example:

```text
important-api
-- get-users
-- create-user
-- static
   -- index.html
   -- main.css
   -- bundle.js
-- Directive.yaml
```

{% hint style="warning" %}
Do not use the static directory for sensitive data such as secrets. Atmo will be gaining a secrets management system in 2021.
{% endhint %}

Since the directory is included in your Bundle, your Runnables can access the files! Atmo will mount the directory as a read-only filesystem that can be accessed using the `file` namespace of the [Runnable API](../runnable-api/introduction.md). For example:

```rust
use suborbital::file;

let indexHtml = file::get_static("index.html");
```

This allows Atmo to serve static sites, access template files, and more!

