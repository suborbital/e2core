use suborbital::runnable::*;
use suborbital::req;
use suborbital::util;

struct ModifyUrl{}

impl Runnable for ModifyUrl {
    fn run(&self, input: Vec<u8>) -> Result<Vec<u8>, RunErr> {
        let body_str = match util::to_string(input) {
            v => if v.is_empty() {
                req::state("url").unwrap_or_default()
            } else {
                v
            }
        };

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
