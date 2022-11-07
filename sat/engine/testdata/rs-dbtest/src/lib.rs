use suborbital::runnable::*;
use suborbital::db;
use suborbital::util;
use suborbital::db::query;
use suborbital::log;
use uuid::Uuid;

struct RsDbtest{}

impl Runnable for RsDbtest {
    fn run(&self, _: Vec<u8>) -> Result<Vec<u8>, RunErr> {
        let uuid = Uuid::new_v4().to_string();

        let mut args: Vec<query::QueryArg> = Vec::new();
        args.push(query::QueryArg::new("uuid", uuid.as_str()));
        args.push(query::QueryArg::new("email", "connor@suborbital.dev"));

        match db::insert("PGInsertUser", args) {
            Ok(_) => log::info("insert successful"),
            Err(e) => {
                return Err(RunErr::new(500, e.message.as_str()))
            }
        };

        let mut args2: Vec<query::QueryArg> = Vec::new();
        args2.push(query::QueryArg::new("uuid", uuid.as_str()));

        match db::update("PGUpdateUserWithUUID", args2.clone()) {
            Ok(rows) => log::info(format!("update: {}", util::to_string(rows).as_str()).as_str()),
            Err(e) => {
                return Err(RunErr::new(500, e.message.as_str()))
            }
        }

        match db::select("PGSelectUserWithUUID", args2.clone()) {
            Ok(result) => log::info(format!("select: {}", util::to_string(result).as_str()).as_str()),
            Err(e) => {
                return Err(RunErr::new(500, e.message.as_str()))
            }
        }

        match db::delete("PGDeleteUserWithUUID", args2.clone()) {
            Ok(rows) => log::info(format!("delete: {}", util::to_string(rows).as_str()).as_str()),
            Err(e) => {
                return Err(RunErr::new(500, e.message.as_str()))
            }
        }

        // this one should fail
        match db::select("PGSelectUserWithUUID", args2.clone()) {
            Ok(result) => {
                let result_str = util::to_string(result);
                if result_str != "[]" {
                    return Err(RunErr::new(500, format!("select should have returning nothing, but didn't, got: {}", result_str).as_str()));
                }

                return Ok(util::to_vec(String::from("all good!")))
            },
            Err(e) => {
                Ok(util::to_vec(e.message))
            }
        }
    }
}


// initialize the runner, do not edit below //
static RUNNABLE: &RsDbtest = &RsDbtest{};

#[no_mangle]
pub extern fn _start() {
    use_runnable(RUNNABLE);
}
