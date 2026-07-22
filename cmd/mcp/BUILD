load("@rules_go//go:def.bzl", "go_binary")
load("@rules_go//go:def.bzl", "go_test")

package(
        default_visibility = ["//visibility:public"],
)

go_binary(
    name = "cel-expr-mcp",
    srcs = ["main.go"],
    deps = [
        "//net/proto2/proto:descriptor_go_proto",
        "@com_github_google_cel_go//cel",
        "//internal/tools",
        "@com_github_modelcontextprotocol_go_sdk//mcp",
        "@org_golang_google_protobuf//proto",
    ],
)

go_test(
    name = "cel-expr-mcp_test",
    srcs = ["main_test.go"],
    library = ":cel-expr-mcp",
    deps = [
        "//net/proto2/proto:descriptor_go_proto",
        "//internal/proto:test_schema_go_proto",
        "//internal/tools",
        "@com_github_modelcontextprotocol_go_sdk//mcp",
        "@org_golang_google_protobuf//proto",
        "@org_golang_google_protobuf//reflect/protodesc",
    ],
)
