load("@com_google_protobuf//bazel:proto_library.bzl", "proto_library")
load("@rules_go//proto:def.bzl", "go_proto_library")
load("@rules_go//go:def.bzl", "go_library")

package(
    default_visibility = ["//visibility:public"],
)

proto_library(
    name = "test_schema_proto",
    srcs = ["test_schema.proto"],
)

go_proto_library(
    name = "test_schema_go_proto",
    proto = ":test_schema_proto",
)

go_library(
    name = "proto",
    srcs = ["test_schema.pb.go"],
    importpath = "github.com/cel-expr/skills/internal/proto",
    deps = [
        "@org_golang_google_protobuf//proto:go_default_library",
        "@org_golang_google_protobuf//reflect/protoreflect:go_default_library",
        "@org_golang_google_protobuf//runtime/protoimpl:go_default_library",
    ]
)
