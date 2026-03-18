load("//tools/build_defs/go:go_library.bzl", "go_library")
load("//tools/build_defs/go:go_test.bzl", "go_test")

go_library(
    name = "gcloud",
    srcs = [
        "builder.go",
        "command.go",
        "config.go",
        "field.go",
        "generate.go",
        "loader.go",
        "method.go",
        "resource.go",
        "service_info.go",
    ],
)

go_test(
    name = "gcloud_test",
    srcs = [
        "builder_test.go",
        "command_test.go",
        "config_test.go",
        "field_test.go",
        "generate_test.go",
        "golden_test.go",
        "method_test.go",
        "resource_test.go",
        "service_info_test.go",
    ],
    library = ":gcloud",
)
