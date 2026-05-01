load("@com_google_protobuf//bazel:proto_library.bzl", "proto_library")
load("@rules_go//go:def.bzl", "go_library")

package(
            default_visibility = ["//visibility:public"],
)

proto_library(
    name = "test_schema_proto",
    srcs = ["test_schema.proto"],
)

go_library(
    name = "proto",
    importpath = "github.com/cel-expr/skills/internal/proto",
    srcs = ["test_schema.pb.go"],
    deps = [
        "@org_golang_google_protobuf//proto",
        "@org_golang_google_protobuf//reflect/protoreflect",
        "@org_golang_google_protobuf//runtime/protoimpl",
    ],
)

alias(
    name = "test_schema_go_proto",
    actual = ":proto",
)
