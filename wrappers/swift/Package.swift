// swift-tools-version:5.1
//TODO ^ find out minimal version

import PackageDescription

let package = Package(
        name: "EduVpnCommon",
        products: [
            .library(
                    name: "EduVpnCommon",
                    targets: ["EduVpnCommon"]),
        ],
        dependencies: [
            .package(path: "CEduVpnCommon"),
        ],
        targets: [
            .target(
                    name: "EduVpnCommon",
                    dependencies: ["CEduVpnCommon"]),
            .testTarget(
                    name: "EduVpnCommonTests",
                    dependencies: ["EduVpnCommon"]),
        ]
)
