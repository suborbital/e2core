use suborbital::runnable::*;
use suborbital::cache;

struct RustSet{}

impl Runnable for RustSet {
    fn run(&self, input: Vec<u8>) -> Result<Vec<u8>, RunErr> {
        cache::set("important", input, 0);
    
        Ok(String::from("hello").as_bytes().to_vec())
    }
}


// initialize the runner, do not edit below //
static RUNNABLE: &RustSet = &RustSet{};

#[no_mangle]
pub extern fn _start() {
    use_runnable(RUNNABLE);
}
