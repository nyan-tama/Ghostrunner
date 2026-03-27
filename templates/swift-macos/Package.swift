// swift-tools-version: 6.0

import PackageDescription

let package = Package(
    name: "{{PROJECT_NAME}}",
    platforms: [
        .macOS(.v14)
    ],
    targets: [
        .executableTarget(
            name: "App",
            path: "Sources/App"
        ),
        .testTarget(
            name: "AppTests",
            dependencies: ["App"],
            path: "Tests/AppTests"
        ),
    ]
)
