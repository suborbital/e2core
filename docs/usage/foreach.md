# ForEach

When building an Atmo request handler or schedule, you can use the `ForEach` clause to iterate over an array of values, executing a Runnable on each element in the array.

```yaml
  - type: request
    resource: /greetings
    method: POST
    steps:
      - fn: set-array
	  	as: names

      - forEach:
          in: names
          fn: hello-name
          as: greetings
```

To use `forEach`, you must have a JSON array of objects in state. A simple array of strings or numbers is not currently supported, but is coming soon. The Runnable is called once for each element in the array, with the element added to the `__elem` state key.

{% hint style="info" %}
In your Runnable's code, use `req::state("__elem")` to get the current array element being handled.
{% endhint %}

The example above takes the request body, saves it to the handler's state, and runs the `hello-name` Runable against each element, which changes all of the values. The result is then saved to the `greetings` key in state:
Input:
```json
[
    {
        "name": "Connor"
    },
    {
        "name": "Jimmy"
    },
    {
        "name": "Bob"
    }
]
```
Output:
```json
[
    {
        "name": "Hello Connor"
    },
    {
        "name": "Hello Jimmy"
    },
    {
        "name": "Hello Bob"
    }
]
```

Here's an example Rust runnable that can be called with `forEach`:
```rust
#[derive(Serialize, Deserialize)]
struct Elem {
    name: String
}

struct HelloName{}

impl Runnable for HelloName {
    fn run(&self, _: Vec<u8>) -> Result<Vec<u8>, RunErr> {
        let elem_json = req::state_raw("__elem");

        let mut elem: Elem = match serde_json::from_slice(elem_json.unwrap_or_default().as_slice()) {
            Ok(e) => e,
            Err(_) => return Err(RunErr::new(500, "failed to from_slice"))
        };

        elem.name = format!("Hello {}", elem.name);

        Ok(serde_json::to_vec(&elem).unwrap_or_default())
    }
}
```