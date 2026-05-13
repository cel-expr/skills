load("@rules_go//go:def.bzl", "go_binary")
load("@rules_go//go:def.bzl", "go_test")

package(
        default_visibility = ["//visibility:public"],
)

go_binary(
    name = "cel-expr-mcp",
    srcs = ["main.go"],
    deps = [
        "//internal/tools",
        "@com_github_modelcontextprotocol_go_sdk//mcp",
    ],
)

go_test(
    name = "cel-expr-mcp_test",
    srcs = ["main_test.go"],
    library = ":cel-expr-mcp",
    deps = [
        "//internal/tools",
        "@com_github_modelcontextprotocol_go_sdk//mcp",
    ],
)
