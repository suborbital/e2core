use suborbital::runnable::*;
use suborbital::req;
use serde_json;
use serde::{Serialize, Deserialize};

#[derive(Serialize, Deserialize)]
struct Elem {
    name: String
}

struct RunEach{}

impl Runnable for RunEach {
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


// initialize the runner, do not edit below //
static RUNNABLE: &RunEach = &RunEach{};

#[no_mangle]
pub extern fn init() {
    use_runnable(RUNNABLE);
}
