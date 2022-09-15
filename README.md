<img width="740" alt="DeltaV" src="https://user-images.githubusercontent.com/5942370/178003243-8dd979b2-b92b-47b3-8a74-a391843b83b3.png">

**Suborbital E2 Core** is a server and SDK that allows developers to add third-party plugins to any application. Plugins are developed using familiar languages like JavaScript, TypeScript, Go, and Rust, and are executed in a securely sandboxed environment. DeltaV can be run within proviate infrastructure and protects against potential malicious untrusted code while providing useful capabilities to plugin developers.

E2 Core is a single statically compiled binary, and can be run on x86 or ARM, containerized or otherwise. It runs as a server, and allows applications to execute plugins using a simple HTTP, RPC, or streaming interface. The admin API makes it simple to manage available plugins, including built-in versioning and namespacing.

Use cases include:
- Running custom logic within an ETL/ELT pipeline
- Adding plugins to streaming platforms like NATS or Kafka/Redpanda
- Allowing users to "write their own webhooks"
- Allowing third-party developers to render custom UI elements

E2 Core pairs with our [Subo CLI](https://github.com/suborbital/subo) for local plugin development and command-line server administration.

Get started with our [guide](https://docs.suborbital.dev/deltav/getting-started), or read the [full documentation](https://docs.suborbital.dev/deltav/reference).