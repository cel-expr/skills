load("@rules_go//go:def.bzl", "go_test")
load("@rules_go//go:def.bzl", "go_library")

package(
            default_visibility = ["//visibility:public"],
)

go_library(
    name = "tools",
    srcs = [
        "compile.go",
        "config.go",
        "coverage.go",
        "evaluate.go",
        "input_schema.go",
        "prompt.go",
    ],
    importpath = "github.com/cel-expr/skills/internal/tools",
    deps = [
        "@com_github_google_cel_go//cel",
        "@com_github_google_cel_go//common/ast",
        "@com_github_google_cel_go//common/env",
        "@com_github_google_cel_go//common/types",
        "@com_github_google_cel_go//common/types/ref",
        "@com_github_google_cel_go//ext",
        "@org_golang_google_protobuf//encoding/protojson",
        "@org_golang_google_protobuf//proto",
    ],
)

go_test(
    name = "tools_test",
    srcs = [
        "compile_test.go",
        "config_test.go",
        "evaluate_test.go",
        "input_schema_test.go",
        "prompt_test.go",
    ],
    data = glob(["testdata/**"]),
    embed = [":tools"],
    deps = [
        "@com_github_google_cel_go//cel",
        "@com_github_google_cel_go//common/env",
        "//internal/proto",
    ],
)
