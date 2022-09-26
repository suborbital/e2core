<img width="883" alt="e2core" src="https://user-images.githubusercontent.com/5942370/190490109-2fd7f923-ba01-4675-a07c-3d571e7c314a.png">

**Suborbital E2 Core** is a server and SDK that allows developers to add third-party plugins to any application. Plugins are developed using familiar languages like JavaScript, TypeScript, Go, and Rust, and are executed in a securely sandboxed environment. E2Core can be run within private infrastructure while protecting against potential malicious untrusted code and providing useful capabilities to plugin developers.

E2 Core is a single statically compiled binary, and can be run on x86 or ARM, containerized or otherwise. It runs as a server, and allows applications to execute plugins using a simple HTTP, RPC, or streaming interface. The admin API makes it simple to manage available plugins, including built-in versioning and namespacing.

Use cases include:
- Running custom logic within an ETL/ELT pipeline
- Adding plugins to streaming platforms like NATS or Kafka/Redpanda
- Allowing users to "write their own webhooks"
- Allowing third-party developers to render custom UI elements

E2 Core pairs with our [Subo CLI](https://github.com/suborbital/subo) for local plugin development and command-line server administration.

**E2 Core is gearing up for its first release, expected in October 2022. This will include extensive documentation and demos, so look out for that!**

### Running locally
If you'd like to run E2 Core locally, you can run `make e2core/install` and then `e2core start ./example-project/modules.wasm.zip`. Plugins can be executed by calling `POST /name/:identifier/:namespace/:name`, for example `curl -d 'world' localhost:8080/com.suborbital.app/default/helloworld-rs`