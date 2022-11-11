import Suborbital

class SwiftGet: Suborbital.Runnable {
    func run(input: String) -> String {
        return Suborbital.CacheGet(key: "important")
    }
}

Suborbital.Set(runnable: SwiftGet())
