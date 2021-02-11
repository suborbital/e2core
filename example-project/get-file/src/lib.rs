use suborbital::runnable;
use suborbital::req;
use suborbital::file;

struct GetFile{}

impl runnable::Runnable for GetFile {
    fn run(&self, _: Vec<u8>) -> Option<Vec<u8>> {
        let filename = req::url_param("file");

        Some(file::get_static(filename.as_str()).unwrap_or("failed".as_bytes().to_vec()))
    }
}


// initialize the runner, do not edit below //
static RUNNABLE: &GetFile = &GetFile{};

#[no_mangle]
pub extern fn init() {
    runnable::set(RUNNABLE);
}
