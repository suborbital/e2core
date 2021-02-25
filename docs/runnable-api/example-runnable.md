# Example Runnable

Here is an example of a Runnable, written in Rust.

{% hint style="info" %}
The `subo` CLI tool will automatically create new Runnables for you with the `subo create runnable` command.
{% endhint %}

```rust
use suborbital::{req, runnable, util};

struct Foobar{}

impl runnable::Runnable for Foobar {
    fn run(&self, _: Vec<u8>) -> Option<Vec<u8>> {
        let body = req::body_raw();
        let body_string = util::to_string(body);
    
        Some(String::from(format!("hello {}", body_string)).as_bytes().to_vec())
    }
}


// initialize the runner, do not edit below //
static RUNNABLE: &Foobar = &Foobar{};

#[no_mangle]
pub extern fn init() {
    runnable::set(RUNNABLE);
}

```

This Runnable uses the `req` namespace to fetch the body of the HTTP request being handled, and then returns it. To learn about all of the Runnable API namespaces, read on!

