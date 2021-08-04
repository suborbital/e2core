# Deploying Atmo

{% hint style="warning" %}
Atmo is still in early Beta, and as such should not yet be used for production workloads.
{% endhint %}

Atmo is distributed as a Docker image: `suborbital/atmo`

To run Atmo, you can mount your Runnable Bundle as a volume, build your own container image that embeds it, or set Atmo to wait for a bundle to be uploaded.

## Volume mount

To mount as a volume:

```text
> docker run -v /path/to/bundle/directory:/home/atmo -e ATMO_HTTP_PORT=8080 -p 8080:8080 suborbital/atmo:latest atmo
```

This will launch Atmo, assign it to listen on port 8080, and run in HTTP mode.

## Embed Bundle

To create your own Docker image with your Bundle embedded, you can use a Dockerfile similar to this:

```yaml
FROM suborbital/atmo:latest

COPY ./runnables.wasm.zip .

ENTRYPOINT atmo
```

Building this Dockerfile would result in an image that doesn't need a volume mount.

## Bundle upload

To upload a bundle after launching Atmo, use the `--wait` flag or set the `ATMO_WAIT=true` env var. This will cause Atmo to check the disk once per second until it finds a bundle rather than exiting with an error if no bundle is found. This method allows you to launch Atmo and then upload a bundle seperately by copying it into the running container, as with the [experimental Kubernetes deployment](https://github.com/suborbital/atmo-k8s-helm).

### HTTPS

To run with HTTPS, replace `ATMO_HTTP_PORT=8080` with `ATMO_DOMAIN=example.com` to enable LetsEncrypt on ports 443 and 80. You will need to pass the `-p` Docker flag for each.

### Logging

To control logging in Atmo, you can use its environment variables:

* `ATMO_LOG_LEVEL` can be set to any of `trace, debug, info, warn, error`
* `ATMO_LOG_FILE` can be set to a file to log to \(stdout will become plaintext logs, structured logs will be written to the file\)

### Schedules

To prevent an Atmo instance from executing the [Schedules](schedules.md) defined in your Directive, you can set the `ATMO_RUN_SCHEDULES=false` env var. This can be useful for running non-idempotent jobs on a specific worker instance.

