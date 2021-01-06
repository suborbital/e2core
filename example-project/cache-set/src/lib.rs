use suborbital::runnable;
use suborbital::request;
use suborbital::cache;
use suborbital::log;

struct CacheSet{}

impl runnable::Runnable for CacheSet {
    fn run(&self, input: Vec<u8>) -> Option<Vec<u8>> {
        let req = request::from_json(input).unwrap();

        let key: &str = req.url.split('/').last().unwrap();
        log::info(format!("setting cache value {}", key).as_str());

        cache::set(key, Vec::from(req.body), 0);

        Some(String::from("ok").as_bytes().to_vec())
    }
}


// initialize the runner, do not edit below //
static RUNNABLE: &CacheSet = &CacheSet{};

#[no_mangle]
pub extern fn init() {
    runnable::set(RUNNABLE);
}
