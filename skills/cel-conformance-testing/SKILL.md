---
name: cel-conformance-testing
description: >-
  Skill for authoring, evaluating, and contributing Common Expression Language (CEL)
  conformance tests using the cel-spec textproto format.
---

# Common Expression Language (CEL) Conformance Testing Skill

Use this skill to author, execute, validate, and publish CEL conformance tests.
Conformance tests define the formal behavioral specification of CEL across
implementations (cel-go, cel-cpp, cel-java).

## Conformance Test Overview & Locations

Conformance tests in CEL are defined in Protocol Buffer text format (`textproto`)
or Protocol Buffer JSON (`protojson`) using the protobuf schema defined in `cel-spec`:
`cel.expr.conformance.test.SimpleTestFile`.

### Repository Location

All standard conformance tests are maintained in the [cel-expr/cel-spec](https://github.com/cel-expr/cel-spec)
repository on GitHub under:
```
tests/simple/testdata/
```

Key test files include:
- `basic.textproto`: Fundamental self-evaluating forms and core operations.
- `comparisons.textproto`: Equality and order comparisons across numeric and complex types.
- `integer_math.textproto`: Arithmetic, overflows, and signed/unsigned math operations.
- `logic.textproto`: Short-circuiting logical operations (`&&`, `||`, `!`).
- `block_ext.textproto`: Scoped variable bindings via `cel.block`.
- `type_deduction.textproto`: Static type deduction and parameterized types.
- `proto2.textproto` & `proto3.textproto`: Protobuf field access, well-known types, and extensions.

Every test file must begin with the protobuf header directives:
```textproto
# proto-file: ../../../proto/cel/expr/conformance/test/simple.proto
# proto-message: cel.expr.conformance.test.SimpleTestFile
```

## Structure & Mapping to Unit Tests

A conformance test file (`SimpleTestFile`) organizes tests into sections:
- `SimpleTestFile`: High-level suite (`name`, `description`, repeated `section`).
- `SimpleTestSection`: Group of related cases (`name`, `description`, repeated `test`).
- `SimpleTest`: Individual test scenario.

### Mapping between Unit Test and Conformance Test Structure

The conformance test structure maps directly to CEL unit testing concepts (such as `cel-go/interpreter/interpreter_test.go`):

| Unit Test Field | `SimpleTest` Mapping | Description |
| --------------- | -------------------- | ----------- |
| `expr` | `expr` | CEL expression under test |
| `container` | `container` | Namespace container for identifier resolution |
| `vars` (`VariableDecl`) | `type_env` entries | Type declarations for variables and functions |
| `in` (`any`) | `bindings` | Input variable bindings (`ExprValue`) |
| `out` (`any`) | `value` | Expected result value (`expr.Value`) |
| `err` (string) | `eval_error` | Expected evaluation error (`ErrorSet`) |
| `unchecked: true` | `disable_check: true` | Bypasses static type check to test runtime evaluation |
| - | `check_only: true` | Validates deduced output type against `typed_result` without evaluation |
| - | `disable_macros: true` | Disables macro expansion during parsing |

### Example Conformance Test

```textproto
name: "policy_conformance"
description: "Conformance tests for policy rule evaluation"
section {
  name: "role_checks"
  test {
    name: "admin_access"
    expr: "user.role == 'admin'"
    type_env {
      name: "user"
      ident {
        type {
          message_type: "example.User"
        }
      }
    }
    bindings {
      key: "user"
      value {
        value {
          object_value {
            [type.googleapis.com/example.User] { role: "admin" }
          }
        }
      }
    }
    value { bool_value: true }
  }
}
```

## What Makes a Good Conformance Test

1. **Targeted & Minimal**:
   - Each test case should exercise exactly one semantic rule or edge condition.
   - Keep expressions as small as possible so test failures isolate the exact flaw.

2. **Language-Agnostic Assertions**:
   - Conformance tests must pass in Go, C++, Python, and Java runtimes (see [Target Runtimes Reference](references/target-runtimes.md)).
   - For errors, use `eval_error` or `any_eval_errors`. Avoid asserting implementation-specific error messages unless verifying standardized error codes.
   - For floating point values, account for IEEE-754 semantics (e.g. `NaN != NaN`).

3. **Thorough Edge Case Coverage**:
   - **Zero-ish and extremes**: `0`, `0u`, `0.0`, `-0.0`, empty strings, empty lists, empty maps, `int64.min`, `int64.max`.
   - **Short-circuiting**: Verify that errors and unknowns in unreachable branches do not prevent evaluation (e.g. `false && (1/0 == 0)` evaluates to `false`).
   - **Type Checking**: Use `check_only: true` and `typed_result { deduced_type { ... } }` to assert static type inference without needing runtime data.
   - **Runtime Dynamic Checks**: Use `disable_check: true` to test dynamic evaluation behavior for undeclared variables or unsupported operations.

4. **Self-Contained & Deterministic**:
   - Explicitly declare all identifiers in `type_env` unless verifying undeclared identifier behavior.
   - Supply all required input bindings in `bindings`.

## Running Conformance Tests

### Using the Suite Runner Script (`run-conformance.sh`)

A dedicated test runner script is provided with this skill at `scripts/run-conformance.sh` to execute the CLI against all or filtered `cel-spec` test suites:

```bash
# Run all cel-spec conformance tests (auto-detects testdata directory):
./skills/cel-conformance-testing/scripts/run-conformance.sh

# Run only a specific test file:
./skills/cel-conformance-testing/scripts/run-conformance.sh --file basic.textproto

# Filter across all suites by test name substring:
./skills/cel-conformance-testing/scripts/run-conformance.sh --filter int_zero

# Specify an explicit testdata directory:
./skills/cel-conformance-testing/scripts/run-conformance.sh --dir /path/to/cel-spec/tests/simple/testdata
```

The script automatically applies standard `cel-go` skip rules, builds the CLI binary, executes tests across each file, and prints a summary report.

### Using the CLI directly (`cel-expr conformance`)

```bash
# Run an entire conformance test file
cel-expr conformance -tests path/to/suite.textproto

# Filter to run a specific test or section
cel-expr conformance -tests path/to/suite.textproto -filter admin_access

# Skip specific tests
cel-expr conformance -tests path/to/suite.textproto -skip prefix1,prefix2

# Provide external protobuf definitions (FileDescriptorSet)
cel-expr conformance -tests path/to/suite.textproto -fds path/to/descriptors.fds

# Run via stdin
cat path/to/suite.textproto | cel-expr conformance -tests -
```

### Using the MCP Tool (`cel_evaluate_conformance`)

Call the `cel_evaluate_conformance` tool with:
```json
{
  "tests": "path/to/tests.textproto",
  "filter": "optional_name_filter"
}
```

Or pass inline textproto or protojson directly as the `tests` argument.

## Publishing Conformance Tests to GitHub `cel-spec`

Follow this workflow to contribute conformance tests to the upstream specification:

1. **Fork and Clone**:
   ```bash
   git clone https://github.com/<your-username>/cel-spec
   cd cel-spec
   git checkout -b add-feature-conformance-tests
   ```

2. **Locate or Create Test Files**:
   - Add tests to an existing category file under `tests/simple/testdata/` (e.g., `tests/simple/testdata/string_ext.textproto`).
   - If adding tests for a new extension or major capability, create a new file named `{feature}.textproto` or `{feature}_ext.textproto`.

3. **Verify Header and Formatting**:
   - Ensure the file begins with the required proto comments:
     ```textproto
     # proto-file: ../../../proto/cel/expr/conformance/test/simple.proto
     # proto-message: cel.expr.conformance.test.SimpleTestFile
     ```
   - Use 2-space indentation.

4. **Validate against Runtimes**:
   - Test locally using `cel-expr conformance`:
     ```bash
     cel-expr conformance -tests tests/simple/testdata/{feature}.textproto
     ```
   - If Bazel is installed in `cel-spec`:
     ```bash
     bazel test //tests/simple:simple_test
     ```
   - Validate across target runtimes as described in [Target Runtimes Reference](references/target-runtimes.md).

5. **Submit a Pull Request**:
   - Commit changes with a descriptive message referencing the relevant specification chapter or issue.
   - Open a Pull Request against `cel-expr/cel-spec:master`.
