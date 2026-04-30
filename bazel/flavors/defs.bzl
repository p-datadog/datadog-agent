# Flavor unit-test tag sets for the Datadog Agent, mirroring tasks/build_tags.py.
#
# FLAVOR_UNIT_TEST_TAGS maps each AgentFlavor name to the tag set used when
# running unit tests for that flavor:
#
#   build_tags[flavor]["unit-tests"].union(COMMON_TAGS)
#
# Use flavor_gotags(flavor_name) to get a gotags-ready value for a go_test rule.
# It handles the Linux-only select() automatically.
#
# To verify this file is in sync with tasks/build_tags.py:
#   bazel test //bazel/flavors:verify_flavor_tags

# LINUX_ONLY_TAGS mirrors LINUX_ONLY_TAGS from tasks/build_tags.py.
# dda inv test never passes these tags on non-Linux platforms.
# Prefer flavor_gotags() over consulting this list directly.
LINUX_ONLY_TAGS = [
    "crio",
    "jetson",
    "linux_bpf",
    "netcgo",
    "nvml",
    "pcap",
    "podman",
    "systemd",
    "trivy",
]

# FLAVOR_UNIT_TEST_TAGS maps each AgentFlavor name to its unit-test tag set.
# Each list includes COMMON_TAGS and the "test" tag.
# Tags from UNIT_TEST_EXCLUDE_TAGS (datadog.no_waf, pcap) are absent.
# Tags are sorted alphabetically.
FLAVOR_UNIT_TEST_TAGS = {
    "base": [
        "cel",
        "clusterchecks",
        "consul",
        "containerd",
        "cri",
        "crio",
        "docker",
        "ec2",
        "etcd",
        "fargateprocess",
        "grpcnotrace",
        "jetson",
        "jmx",
        "kubeapiserver",
        "kubelet",
        "ncm",
        "netcgo",
        "no_dynamic_plugins",
        "nvml",
        "oracle",
        "orchestrator",
        "otlp",
        "podman",
        "python",
        "retrynotrace",
        "sharedlibrarycheck",
        "systemd",
        "systemprobechecks",
        "test",
        "trivy",
        "trivy_no_javadb",
        "zk",
        "zlib",
        "zstd",
    ],
    "fips": [
        "cel",
        "consul",
        "containerd",
        "cri",
        "crio",
        "docker",
        "ec2",
        "etcd",
        "fargateprocess",
        "goexperiment.systemcrypto",
        "grpcnotrace",
        "jetson",
        "jmx",
        "kubeapiserver",
        "kubelet",
        "ncm",
        "netcgo",
        "no_dynamic_plugins",
        "nvml",
        "oracle",
        "orchestrator",
        "otlp",
        "podman",
        "python",
        "requirefips",
        "retrynotrace",
        "sharedlibrarycheck",
        "systemd",
        "systemprobechecks",
        "test",
        "trivy",
        "trivy_no_javadb",
        "zk",
        "zlib",
        "zstd",
    ],
    "heroku": [
        "bundle_installer",
        "consul",
        "etcd",
        "grpcnotrace",
        "jmx",
        "ncm",
        "netcgo",
        "no_dynamic_plugins",
        "otlp",
        "python",
        "retrynotrace",
        "sharedlibrarycheck",
        "systemprobechecks",
        "test",
        "trivy_no_javadb",
        "zk",
        "zlib",
        "zstd",
    ],
    "iot": [
        "grpcnotrace",
        "jetson",
        "no_dynamic_plugins",
        "retrynotrace",
        "systemd",
        "test",
        "trivy_no_javadb",
        "zlib",
        "zstd",
    ],
    "dogstatsd": [
        "containerd",
        "docker",
        "grpcnotrace",
        "kubelet",
        "no_dynamic_plugins",
        "podman",
        "retrynotrace",
        "test",
        "trivy_no_javadb",
        "zlib",
        "zstd",
    ],
}

def flavor_gotags(flavor_name):
    """Returns the gotags value for a go_test rule for the given flavor.

    Tags in LINUX_ONLY_TAGS are wrapped in a select() so they are only active
    on Linux, matching the behaviour of dda inv test.
    """
    tags = FLAVOR_UNIT_TEST_TAGS[flavor_name]
    always = [t for t in tags if t not in LINUX_ONLY_TAGS]
    linux_only = [t for t in tags if t in LINUX_ONLY_TAGS]
    return always + select({
        "@platforms//os:linux": linux_only,
        "//conditions:default": [],
    })
