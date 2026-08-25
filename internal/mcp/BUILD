load("@rules_go//go:def.bzl", "go_library", "go_test")

package(
    default_visibility = ["//visibility:public"],
)

go_library(
    name = "mcp",
    srcs = ["server.go"],
    importpath = "github.com/cel-expr/skills/internal/mcp",
    deps = [
        "//internal/tools",
        "@com_github_modelcontextprotocol_go_sdk//mcp",
        "@dev_cel_cel_go//cel",
        "@org_golang_google_protobuf//proto",
        "@org_golang_google_protobuf//types/descriptorpb",
    ],
)

go_test(
    name = "mcp_test",
    srcs = ["server_test.go"],
    embed = [":mcp"],
    deps = [
        "//internal/proto:test_schema_go_proto",
        "//internal/tools",
        "@com_github_modelcontextprotocol_go_sdk//mcp",
        "@dev_cel_cel_go//cel",
        "@org_golang_google_protobuf//proto",
        "@org_golang_google_protobuf//reflect/protodesc",
        "@org_golang_google_protobuf//types/descriptorpb",
    ],
)
