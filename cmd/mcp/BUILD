load("@rules_go//go:def.bzl", "go_binary")

package(
    default_visibility = ["//visibility:public"],
)

go_binary(
    name = "cel-expr-mcp",
    srcs = ["main.go"],
    deps = [
        "//internal/mcp",
        "@com_github_modelcontextprotocol_go_sdk//mcp",
    ],
)
