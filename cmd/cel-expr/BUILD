load("@rules_go//go:def.bzl", "go_binary", "go_test")

package(
    default_visibility = ["//visibility:public"],
)

go_binary(
    name = "cel-expr",
    srcs = ["main.go"],
    deps = [
        "//internal/mcp",
        "//internal/tools",
        "@com_github_modelcontextprotocol_go_sdk//mcp",
        "@dev_cel_cel_go//cel",
    ],
)

go_test(
    name = "cel-expr_test",
    srcs = [
        "main.go",
        "main_test.go",
    ],
    deps = [
        "//internal/mcp",
        "//internal/proto:test_schema_go_proto",
        "//internal/tools",
        "@com_github_modelcontextprotocol_go_sdk//mcp",
        "@dev_cel_cel_go//cel",
        "@org_golang_google_protobuf//proto",
        "@org_golang_google_protobuf//reflect/protodesc",
        "@org_golang_google_protobuf//types/descriptorpb",
    ],
)
