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
        "//third_party/golang/github_com/google/jsonschema_go/v/v0/jsonschema",
        "@com_github_modelcontextprotocol_go_sdk//mcp",
    ],
)

go_test(
    name = "cel-expr-mcp_test",
    srcs = [
        "main.go",
        "main_test.go",
    ],
    deps = [
        "@com_github_google_cel_go//cel",
        "//internal/tools",
        "@com_github_modelcontextprotocol_go_sdk//mcp",
    ],
)
