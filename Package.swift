// swift-tools-version: 6.1
// The swift-tools-version declares the minimum version of Swift required to build this package.

// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import PackageDescription

let package = Package(
    name: "TestServer",
    platforms: [
        .macOS(.v12) // Matches your current toolchain requirement
    ],
    products: [
        .library(
            name: "TestServer",
            targets: ["TestServer"]
        ),
    ],
    targets: [
        .target(
            name: "TestServer",
            dependencies: [],
            // Point to the subdirectory containing your wrapper code
            path: "sdks/swift/Sources/TestServer"
        ),
        .testTarget(
            name: "TestServerTests",
            dependencies: ["TestServer"],
            // Point to the subdirectory containing your verification tests
            path: "sdks/swift/Tests/TestServerTests"
        ),
    ]
)
