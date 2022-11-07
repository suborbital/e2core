use suborbital::runnable::*;
use suborbital::req;

struct RustUrlquery{}

impl Runnable for RustUrlquery {
    fn run(&self, _: Vec<u8>) -> Result<Vec<u8>, RunErr> {
        let query = req::query_param("message");
    
        Ok(String::from(format!("hello {}", query)).as_bytes().to_vec())
    }
}


// initialize the runner, do not edit below //
static RUNNABLE: &RustUrlquery = &RustUrlquery{};

#[no_mangle]
pub extern fn _start() {
    use_runnable(RUNNABLE);
}
