use suborbital::runnable;
use suborbital::req;
use suborbital::util;

struct ModifyUrl{}

impl runnable::Runnable for ModifyUrl {
    fn run(&self, _: Vec<u8>) -> Option<Vec<u8>> {
        let body_str = util::to_string(req::body_raw());

        let modified = format!("{}/suborbital", body_str.as_str());
        Some(modified.as_bytes().to_vec())
    }
}


// initialize the runner, do not edit below //
static RUNNABLE: &ModifyUrl = &ModifyUrl{};

#[no_mangle]
pub extern fn init() {
    runnable::set(RUNNABLE);
}
