load("@com_google_protobuf//bazel:proto_library.bzl", "proto_library")
load("@rules_go//proto:def.bzl", "go_proto_library")

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
