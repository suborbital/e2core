# Connections

In order to build a useful application, Atmo needs to be able to connect to external sources. Currently, the only connection type is [NATS](https://nats.io/), but upcoming releases will include additional types such as Redis, databases, and more.

To create a connection, add a `connections` section to your Directive:

```yaml
connections:
  nats:
    serverAddress: nats://localhost:4222
```

When Atmo starts up, it will create a connection to the NATS server and make it available as a stream source:

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

