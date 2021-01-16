import Suborbital

class CacheSet: Suborbital.Runnable {
    func run(input: String) -> String {
        let key = Suborbital.ReqParam(key: "key")
        let body = Suborbital.ReqBodyRaw()

        Suborbital.LogInfo(msg: "setting cache value \(key): \(body)")

        Suborbital.CacheSet(key: key, value: body, ttl: 0)

        return ""
    }
}

@_cdecl("init")
func `init`() {
    Suborbital.Set(runnable: CacheSet())
}