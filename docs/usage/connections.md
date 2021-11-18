# Connections

In order to build a useful application, Atmo needs to be able to connect to external resources. Currently, Atmo can connect to [NATS](https://nats.io/) and [Redis](https://redis.io/), and upcoming releases will include additional types such as databases and more.

To create connections, add a `connections` section to your Directive. When Atmo starts up, it will establish the connections you've configured, and make them available to your application in a few different ways.

## Stream sources
There are two available stream sources (NATS and Kafka) that can be used as sources for your handlers:
```yaml
connections:
  nats:
    serverAddress: nats://localhost:4222
  kafka:
    brokerAddress: localhost:9092
```

The NATS or Kafka connection is made available as a stream source:

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

## Data sources
SQL databases and caches can be connected to Atmo to be made available to your Runnables using the Runnable API:
```yaml
connections:
  database:
    type: postgresql
    connectionString: env(DATABASE)
  redis:
    serverAddress: localhost:6379
```
SQL database connections of type `mysql` and `postgresql` are available, and they are discussed in detail in the [next section](./using-sql-databases.md).

Redis connections are made available to Runnables utilizing the `cache` capability:
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