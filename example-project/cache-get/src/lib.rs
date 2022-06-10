use suborbital::runnable::*;
use suborbital::req;
use suborbital::cache;
use suborbital::log;

struct CacheGet{}

impl Runnable for CacheGet {
    fn run(&self, _: Vec<u8>) -> Result<Vec<u8>, RunErr> {
        let key = req::url_param("key");

        log::info(format!("getting cache value {}", key).as_str());

        let val = cache::get(key.as_str()).unwrap_or_default();
    
        Ok(val)
    }
}


// initialize the runner, do not edit below //
static RUNNABLE: &CacheGet = &CacheGet{};

#[no_mangle]
pub extern fn _start() {
    use_runnable(RUNNABLE);
}
