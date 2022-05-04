# ConstD, the constellation manager
> `constd` is an experiment built on two other experiments, so you should be doubly afraid of using it in production. That said, we'd love your feedback! 🙂

Sat is designed to be run in a constellation, with many instances collaborating to execute a higher-level application. Specifically, constellations can run [Atmo](https://github.com/suborbital/atmo) projects, which are comprised of several functions that coordinate to create a server application. Since these projects are [declarative](https://atmo.suborbital.dev/concepts/the-directive), it is possible to distribute the app's compute and let the constellation 'figure it out'. That's the goal of `constd`.

As mentioned up top, `constd` is an experiment built on two other experiments:
- Sat, a small and fast WebAssembly server
- Atmo proxy mode, designed to mesh with a constellation

## Build and run constd
> You'll need Go and Docker to run Sat and ConstD, and you'll need to clone the [Sat](https://github.com/suborbital/sat) and [Atmo](https://github.com/suborbital/atmo) repos.

To get started, build Atmo proxy. In the Atmo repo, run:
```bash
make docker/dev/proxy
```
This builds the `suborbital/atmo-proxy:dev` Docker image.

Next, in the Sat repo, build `constd` and start it:
```bash
make constd

.bin/constd {absolute/path/to}/atmo/example-project/runnables.wasm.zip
```
`constd` will launch `atmo-proxy` and a constellation of Sat instances. Make a request to test it:
```bash
curl localhost:8080/hello -d 'my friend'
```
The `atmo-proxy` container receives the request, and proxies execution of the WebAssembly functions to the Sat constellation.

Currently, the following features normally found in an Atmo project won't work very well:
- Access to cache
- Authentication for HTTP/GraphQL requests
- Access to static files

But these will come in time!