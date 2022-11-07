use suborbital::runnable::*;
use suborbital::cache;

struct RustGet{}

impl Runnable for RustGet {
    fn run(&self, _: Vec<u8>) -> Result<Vec<u8>, RunErr> {
        let cache_val = cache::get("name").unwrap_or_default();
    
        Ok(cache_val)
    }
}


// initialize the runner, do not edit below //
static RUNNABLE: &RustGet = &RustGet{};

#[no_mangle]
pub extern fn _start() {
    use_runnable(RUNNABLE);
}
