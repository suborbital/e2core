use suborbital::runnable;
use suborbital::req;
use suborbital::util;

struct HelloworldRs{}

impl runnable::Runnable for HelloworldRs {
    fn run(&self, _: Vec<u8>) -> Option<Vec<u8>> {
        let msg = format!("hello {}", util::to_string(req::body_raw()));

        Some(util::to_vec(String::from(msg)))
    }
}


// initialize the runner, do not edit below //
static RUNNABLE: &HelloworldRs = &HelloworldRs{};

#[no_mangle]
pub extern fn init() {
    runnable::set(RUNNABLE);
}
