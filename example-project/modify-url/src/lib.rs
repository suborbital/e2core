use suborbital::runnable::*;
use suborbital::req;
use suborbital::util;

struct ModifyUrl{}

impl Runnable for ModifyUrl {
    fn run(&self, _: Vec<u8>) -> Result<Vec<u8>, RunErr> {
        let body_str = util::to_string(req::body_raw());

        let modified = format!("{}/suborbital", body_str.as_str());
        Ok(modified.as_bytes().to_vec())
    }
}


// initialize the runner, do not edit below //
static RUNNABLE: &ModifyUrl = &ModifyUrl{};

#[no_mangle]
pub extern fn init() {
    use_runnable(RUNNABLE);
}
