use suborbital::runnable::*;
use suborbital::{resp};

struct SetArray{}

impl Runnable for SetArray {
    fn run(&self, body: Vec<u8>) -> Result<Vec<u8>, RunErr> {
        resp::content_type("application/json");

        // returning the request body so it gets set in state
        Ok(body)
    }
}


// initialize the runner, do not edit below //
static RUNNABLE: &SetArray = &SetArray{};

#[no_mangle]
pub extern fn _start() {
    use_runnable(RUNNABLE);
}
