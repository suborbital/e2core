// swift-tools-version:5.3
// The swift-tools-version declares the minimum version of Swift required to build this package.

import PackageDescription

let package = Package(
    name: "cache-set",
    dependencies: [
        .package(name: "Suborbital", url: "https://github.com/suborbital/reactr.git", from: "0.6.3")
    ],
    targets: [
        // Targets are the basic building blocks of a package. A target can define a module or a test suite.
        // Targets can depend on other targets in this package, and on products in packages this package depends on.
        .target(
            name: "cache-set",
            dependencies: ["Suborbital"])
    ]
)
