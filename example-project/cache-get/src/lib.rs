use suborbital::runnable;
use suborbital::req;
use suborbital::cache;
use suborbital::log;

struct CacheGet{}

impl runnable::Runnable for CacheGet {
    fn run(&self, _: Vec<u8>) -> Option<Vec<u8>> {
        let key = req::url_param("key");

        log::info(format!("getting cache value {}", key).as_str());

        let val = cache::get(key.as_str()).unwrap_or_default();
    
        Some(val)
    }
}


// initialize the runner, do not edit below //
static RUNNABLE: &CacheGet = &CacheGet{};

#[no_mangle]
pub extern fn init() {
    runnable::set(RUNNABLE);
}
