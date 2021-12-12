use suborbital::runnable::*;
use suborbital::req;
use suborbital::util;

struct ModifyUrl{}

impl Runnable for ModifyUrl {
    fn run(&self, _: Vec<u8>) -> Result<Vec<u8>, RunErr> {
        let mut url = req::body_raw();

        match req::state("url") {
            Some(val) => {
                if !val.is_empty() {
                    url = util::to_vec(val)
                }
            }
            None => {}
        };

        let modified = format!("{}/suborbital", util::to_string(url));

        Ok(modified.as_bytes().to_vec())
    }
}


// initialize the runner, do not edit below //
static RUNNABLE: &ModifyUrl = &ModifyUrl{};

#[no_mangle]
pub extern fn init() {
    use_runnable(RUNNABLE);
}
