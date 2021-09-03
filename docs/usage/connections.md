# Connections

In order to build a useful application, Atmo needs to be able to connect to external resources. Currently, Atmo can connect to [NATS](https://nats.io/) and [Redis](https://redis.io/), and upcoming releases will include additional types such as databases and more.

To create connections, add a `connections` section to your Directive:

```yaml
connections:
  nats:
    serverAddress: nats://localhost:4222
  redis:
    serverAddress: localhost:6379
```

When Atmo starts up, it will establish the connections you've configured, and make them available to your application in a few different ways.

The NATS connection is made available as a stream source:

```yaml
  - type: stream
    source: nats
    resource: user.created
    steps:
      - fn: record-signup
```

By setting the `source` field of the handler, we tell Atmo to listen to that particular connection and handle messages it sends us. The `resource` field dictates which topic or subject the handler is listening to, which is useful for messaging systems such as NATS and Kafka.

Streams that use an external source can also use the `respondTo` field to set which topic or subject the response message is sent to:

```yaml
- type: stream
    source: nats
    resource: user.login
    steps:
      - fn: record-login
    respondTo: user.send-login-email
```

The Redis connection will be made available to Runnables utilizing the `cache` capability:
```rust
use suborbital::runnable::*;
use suborbital::req;
use suborbital::cache;

struct CacheGet{}

impl Runnable for CacheGet {
    fn run(&self, _: Vec<u8>) -> Result<Vec<u8>, RunErr> {
        let key = req::url_param("key");

        let val = cache::get(key.as_str()).unwrap_or_default();
    
        Ok(val)
    }
}
```