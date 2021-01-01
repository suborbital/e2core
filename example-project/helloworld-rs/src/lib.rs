use suborbital::runnable;
use suborbital::request;

struct HelloworldRs{}

impl runnable::Runnable for HelloworldRs {
    fn run(&self, input: Vec<u8>) -> Option<Vec<u8>> {
        let req = match request::from_json(input) {
            Some(r) => r,
            None => return Some(String::from("failed").as_bytes().to_vec())
        };
    
        Some(String::from(format!("hello {}", req.body)).as_bytes().to_vec())
    }
}


// initialize the runner, do not edit below //
static RUNNABLE: &HelloworldRs = &HelloworldRs{};

#[no_mangle]
pub extern fn init() {
    runnable::set(RUNNABLE);
}
