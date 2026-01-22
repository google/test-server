// swift-tools-version: 6.1
// The swift-tools-version declares the minimum version of Swift required to build this package.

import PackageDescription

let package = Package(
    name: "TestServer",
    platforms: [
        .macOS(.v12)
    ],
    products: [
        .library(name: "TestServer", targets: ["TestServer"])
    ],
    targets: [
        .target(
            name: "TestServer",
            dependencies: []
        ),
        .testTarget(
            name: "TestServerTests",
            dependencies: ["TestServer"]
        ),
    ]
)