load("@rules_go//go:def.bzl", "go_test")
load("@rules_go//go:def.bzl", "go_library")

package(
            default_visibility = ["//visibility:public"],
)

go_library(
    name = "tools",
    importpath = "github.com/cel-expr/skills/internal/tools",
    srcs = [
        "compile.go",
        "config.go",
        "coverage.go",
        "evaluate.go",
        "input_schema.go",
        "prompt.go",
    ],
    deps = [
        "@dev_cel_cel_go//cel",
        "@dev_cel_cel_go//common/ast",
        "@dev_cel_cel_go//common/env",
        "@dev_cel_cel_go//common/types",
        "@dev_cel_cel_go//common/types/ref",
        "@dev_cel_cel_go//ext",
        "@org_golang_google_protobuf//encoding/protojson",
        "@org_golang_google_protobuf//proto",
    ],
)

go_test(
    name = "tools_test",
    srcs = [
        "compile_test.go",
        "config_test.go",
        "coverage_test.go",
        "evaluate_test.go",
        "input_schema_test.go",
        "prompt_test.go",
    ],
    data = glob(["testdata/**"]),
    embed = [":tools"],
    deps = [
        "@dev_cel_cel_go//cel",
        "@dev_cel_cel_go//common/ast",
        "@dev_cel_cel_go//common/env",
        "@dev_cel_cel_go//common/types",
        "//internal/proto:test_schema_go_proto",
    ],
)
