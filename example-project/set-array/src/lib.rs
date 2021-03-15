use suborbital::runnable::*;
use suborbital::{req, resp};

struct SetArray{}

impl Runnable for SetArray {
    fn run(&self, _: Vec<u8>) -> Result<Vec<u8>, RunErr> {
        resp::content_type("application/json");

        // returning the request body so it gets set in state
        Ok(req::body_raw())
    }
}


// initialize the runner, do not edit below //
static RUNNABLE: &SetArray = &SetArray{};

#[no_mangle]
pub extern fn init() {
    use_runnable(RUNNABLE);
}
