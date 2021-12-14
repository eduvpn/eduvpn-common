// swift-tools-version:5.1

import PackageDescription

let package = Package(
        name: "CEduVpnCommon",
        products: [
            .library(name: "CEduVpnCommon", targets: ["CEduVpnCommon"]),
        ],
        targets: [
            .systemLibrary(name: "CEduVpnCommon"),
        ]
)
