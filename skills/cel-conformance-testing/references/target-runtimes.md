# Running Conformance Tests across Target Runtimes

CEL conformance tests define the behavioral specification of CEL and are validated against all core runtime implementations. Before contributing or proposing new conformance test cases to `cel-spec`, verify them across the respective language targets:

## Go (`cel-go`)

Tests can be evaluated using the local runner tooling provided in this repository or via Bazel in the [`cel-go`](https://github.com/cel-expr/cel-go) repository:

- **Via local runner script** (using the `cel-expr` CLI against your local `cel-spec` directory):
  ```bash
  ./skills/cel-conformance-testing/scripts/run-conformance.sh --file <test_file>.textproto
  ```
- **Via Bazel in `cel-go`**:
  ```bash
  # Run standard conformance test suite
  bazel test //conformance:conformance

  # Run the full dashboard conformance suite
  bazel test //conformance:conformance_dashboard
  ```

## C++ (`cel-cpp`)

In the [`cel-cpp`](https://github.com/cel-expr/cel-cpp) repository, conformance tests are executed via Bazel:

```bash
# Run checked conformance tests
bazel test //conformance:conformance_checked

# Run all conformance test targets
bazel test //conformance:...
```

## Java (`cel-java`)

In the [`cel-java`](https://github.com/cel-expr/cel-java) repository, conformance tests are executed via Bazel:

```bash
# Run the primary conformance test suite
bazel test //conformance/src/test/java/dev/cel/conformance:all

# Run all conformance test packages
bazel test //conformance/...
```

## Python (`cel-python`)

In [`cel-python`](https://github.com/cloud-custodian/cel-python), conformance tests are translated from `cel-spec` `.textproto` files into Gherkin feature scenarios using `tools/gherkinize.py` and run via `behave` or `pytest`:

```bash
# Regenerate feature files from cel-spec textprotos
cd features && make all

# Run conformance scenarios
behave

# Or run via pytest
pytest
```

Also, in [`cel-expr/cel-python`](https://github.com/cel-expr/cel-python/blob/main/conformance/BUILD):

```bash
bazel test //conformance:conformance_test
```

