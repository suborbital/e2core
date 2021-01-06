use suborbital::runnable;
use suborbital::request;
use suborbital::cache;
use suborbital::log;

struct CacheGet{}

impl runnable::Runnable for CacheGet {
    fn run(&self, input: Vec<u8>) -> Option<Vec<u8>> {
        let req = request::from_json(input).unwrap();

        let key: &str = req.url.split('/').last().unwrap();
        log::info(format!("getting cache value {}", key).as_str());

        let val = cache::get(key).unwrap_or_default();
    
        Some(val)
    }
}


// initialize the runner, do not edit below //
static RUNNABLE: &CacheGet = &CacheGet{};

#[no_mangle]
pub extern fn init() {
    runnable::set(RUNNABLE);
}
