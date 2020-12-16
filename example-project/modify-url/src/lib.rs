use suborbital::runnable;
use suborbital::request;

struct ModifyUrl{}

impl runnable::Runnable for ModifyUrl {
    fn run(&self, input: Vec<u8>) -> Option<Vec<u8>> {
        let req = match request::from_json(input) {
            Some(r) => r,
            None => return Some(String::from("failed").as_bytes().to_vec())
        };

        let modified = format!("{}/suborbital", req.body.as_str());
        Some(modified.as_bytes().to_vec())
    }
}


// initialize the runner, do not edit below //
static RUNNABLE: &ModifyUrl = &ModifyUrl{};

#[no_mangle]
pub extern fn init() {
    runnable::set(RUNNABLE);
}
