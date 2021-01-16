use suborbital::runnable;
use suborbital::req;
use suborbital::cache;
use suborbital::log;

struct CacheSet{}

impl runnable::Runnable for CacheSet {
    fn run(&self, _: Vec<u8>) -> Option<Vec<u8>> {

        let key = req::url_param("key");
        log::info(format!("setting cache value {}", key).as_str());

        cache::set(key.as_str(), Vec::from(req::body_raw()), 0);

        Some(String::from("ok").as_bytes().to_vec())
    }
}


// initialize the runner, do not edit below //
static RUNNABLE: &CacheSet = &CacheSet{};

#[no_mangle]
pub extern fn init() {
    runnable::set(RUNNABLE);
}
