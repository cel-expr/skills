load("@rules_go//go:def.bzl", "go_binary")
load("@rules_go//go:def.bzl", "go_test")

package(
        default_visibility = ["//visibility:public"],
)

go_binary(
    name = "mcp",
    srcs = ["main.go"],
    deps = [
        "//internal/tools",
        "@com_github_modelcontextprotocol_go_sdk//mcp",
    ],
)

go_test(
    name = "mcp_test",
    srcs = [
      "main.go", 
      "main_test.go",
    ],
    deps = [
        "//internal/tools",
        "@com_github_modelcontextprotocol_go_sdk//mcp",
    ],
)
