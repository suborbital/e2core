use suborbital::runnable::*;
use suborbital::req;
use suborbital::file;

struct GetFile{}

impl Runnable for GetFile {
    fn run(&self, _: Vec<u8>) -> Result<Vec<u8>, RunErr> {
        let mut filename = req::url_param("file");

        if filename == "" {
            filename = req::state("file").unwrap_or_default();
        }

        Ok(file::get_static(filename.as_str()).unwrap_or("failed".as_bytes().to_vec()))
    }
}


// initialize the runner, do not edit below //
static RUNNABLE: &GetFile = &GetFile{};

#[no_mangle]
pub extern fn init() {
    use_runnable(RUNNABLE);
}
