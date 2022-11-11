# Add a new rust function: `HelloWorld`

## Prerequisites

You need to install the `subo` cli.

### Subo Linux installation

```bash
git clone https://github.com/suborbital/subo
cd subo
make subo
```
> For more details, see https://github.com/suborbital/subo

## Create the function skeleton

```bash
subo create runnable hello-world --dir ./examples --lang rust
````

If needed, add your dependencies to the `Cargo.toml` file. Like below (I need `serde` dependency to parse JSON string):

```toml
[dependencies]
suborbital = '0.12.0'
serde = { version = "1.0", features = ["derive"] }
serde_json = "1.0"
```

## Build the function

```bash
subo build ./examples/hello-world
```

## Use the function with Sat

### First build & run Sat

```bash
make sat
SAT_HTTP_PORT=8080 .bin/sat ./examples/hello-world/hello-world.wasm 
```

Now, **Sat** is serving the function on `localhost:8080`

### Call the function with curl

```bash
data='{"text":"from Bob Morane"}'
curl -d "${data}" \
    -H "Content-Type: application/json" \
    -X POST "http://localhost:8080"
```
