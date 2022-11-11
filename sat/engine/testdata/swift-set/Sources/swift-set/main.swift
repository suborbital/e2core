import Suborbital
import Foundation

class SwiftSet: Suborbital.Runnable {
    func run(input: String) -> String {
        Suborbital.CacheSet(key: "name", value: input, ttl: 0)

        return "hello"
    }
}

Suborbital.Set(runnable: SwiftSet())
