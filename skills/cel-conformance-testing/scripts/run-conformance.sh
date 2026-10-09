#!/usr/bin/env bash
# Copyright 2026 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"

# Default skip list matching cel-go/conformance/BUILD.bazel
DEFAULT_SKIPS=(
  "fields/qualified_identifier_resolution/map_key_float"
  "fields/qualified_identifier_resolution/map_key_null"
  "fields/qualified_identifier_resolution/map_value_repeat_key"
  "fields/qualified_identifier_resolution/map_value_repeat_key_heterogeneous"
  "timestamps/duration_converters/get_milliseconds"
  "optionals/optionals/map_optional_select_has"
  "string_ext/value_errors/indexof_out_of_range"
  "string_ext/value_errors/lastindexof_out_of_range"
  "enums/strong_proto2"
  "enums/strong_proto3"
  "type_deductions/wrappers/wrapper_promotion_2"
  "type_deductions/legacy_nullable_types/null_assignable_to_message_parameter_candidate"
  "type_deductions/legacy_nullable_types/null_assignable_to_duration_parameter_candidate"
  "type_deductions/legacy_nullable_types/null_assignable_to_timestamp_parameter_candidate"
  "type_deductions/legacy_nullable_types/null_assignable_to_abstract_parameter_candidate"
)
JOINED_DEFAULT_SKIPS=$(IFS=,; echo "${DEFAULT_SKIPS[*]}")

usage() {
  cat <<EOF
Usage: $(basename "$0") [options] [file...]

Evaluates CEL conformance tests from cel-spec against cel-go using the
cel-expr CLI conformance target.

Options:
  -d, --dir <path>       Directory containing cel-spec .textproto files
                         (default: auto-detected from GOPATH or ../cel-spec)
  -f, --filter <str>     Filter test cases by name substring
  -s, --skip <str>       Test prefixes to skip (comma-separated)
      --file <name>      Run only a specific test file (e.g. basic.textproto)
  -b, --bin <path>       Path to pre-built cel-expr CLI binary
                         (default: builds a temporary binary via 'go build')
  -v, --verbose          Print verbose JSON output for each test file
  -h, --help             Show this help message and exit

Examples:
  # Run all conformance tests:
  $(basename "$0")

  # Run only basic.textproto:
  $(basename "$0") --file basic.textproto

  # Filter to a specific test name across all suites:
  $(basename "$0") --filter int_zero

  # Specify a custom cel-spec testdata directory:
  $(basename "$0") --dir /path/to/cel-spec/tests/simple/testdata
EOF
}

# Locate testdata directory
find_testdata_dir() {
  if [[ -n "${CEL_SPEC_TESTDATA:-}" && -d "${CEL_SPEC_TESTDATA}" ]]; then
    echo "${CEL_SPEC_TESTDATA}"
    return
  fi
  if [[ -n "${CEL_SPEC_DIR:-}" && -d "${CEL_SPEC_DIR}/tests/simple/testdata" ]]; then
    echo "${CEL_SPEC_DIR}/tests/simple/testdata"
    return
  fi

  local candidates=(
    "$(go env GOPATH 2>/dev/null || echo '')/src/github.com/cel-expr/cel-spec/tests/simple/testdata"
    "${REPO_ROOT}/../cel-spec/tests/simple/testdata"
    "${REPO_ROOT}/../../cel-expr/cel-spec/tests/simple/testdata"
  )

  for dir in "${candidates[@]}"; do
    if [[ -n "${dir}" && -d "${dir}" ]]; then
      echo "${dir}"
      return
    fi
  done

  echo ""
}

TESTDATA_DIR=""
FILTER=""
ADDITIONAL_SKIPS=""
SPECIFIC_FILE=""
CLI_BIN=""
VERBOSE=false

while [[ $# -gt 0 ]]; do
  case "$1" in
    -d|--dir)
      TESTDATA_DIR="$2"
      shift 2
      ;;
    -f|--filter)
      FILTER="$2"
      shift 2
      ;;
    -s|--skip)
      ADDITIONAL_SKIPS="$2"
      shift 2
      ;;
    --file)
      SPECIFIC_FILE="$2"
      shift 2
      ;;
    -b|--bin)
      CLI_BIN="$2"
      shift 2
      ;;
    -v|--verbose)
      VERBOSE=true
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      if [[ -f "$1" ]]; then
        SPECIFIC_FILE="$1"
      elif [[ -d "$1" ]]; then
        TESTDATA_DIR="$1"
      else
        echo "Unknown option or argument: $1" >&2
        usage
        exit 1
      fi
      shift
      ;;
  esac
done

if [[ -z "${TESTDATA_DIR}" ]]; then
  TESTDATA_DIR="$(find_testdata_dir)"
fi

if [[ -z "${TESTDATA_DIR}" && -z "${SPECIFIC_FILE}" ]]; then
  echo "Error: cel-spec testdata directory could not be located." >&2
  echo "Please specify with -d / --dir <path> or set CEL_SPEC_TESTDATA." >&2
  exit 1
fi

# Build cel-expr CLI binary if not provided
TMP_DIR=""
if [[ -z "${CLI_BIN}" ]]; then
  TMP_DIR="$(mktemp -d)"
  trap 'rm -rf "${TMP_DIR}"' EXIT
  CLI_BIN="${TMP_DIR}/cel-expr"
  echo "Building cel-expr CLI binary..."
  (cd "${REPO_ROOT}" && go build -o "${CLI_BIN}" ./cmd/cli)
elif [[ ! -x "${CLI_BIN}" ]]; then
  echo "Error: Specified binary '${CLI_BIN}' does not exist or is not executable." >&2
  exit 1
fi

# Combine skip list
ALL_SKIPS="${JOINED_DEFAULT_SKIPS}"
if [[ -n "${ADDITIONAL_SKIPS}" ]]; then
  ALL_SKIPS="${ALL_SKIPS},${ADDITIONAL_SKIPS}"
fi

# Collect files to test
FILES=()
if [[ -n "${SPECIFIC_FILE}" ]]; then
  if [[ -f "${SPECIFIC_FILE}" ]]; then
    FILES+=("${SPECIFIC_FILE}")
  elif [[ -f "${TESTDATA_DIR}/${SPECIFIC_FILE}" ]]; then
    FILES+=("${TESTDATA_DIR}/${SPECIFIC_FILE}")
  else
    echo "Error: Test file '${SPECIFIC_FILE}' not found." >&2
    exit 1
  fi
else
  while IFS= read -r f; do
    FILES+=("$f")
  done < <(find "${TESTDATA_DIR}" -maxdepth 1 -name "*.textproto" | sort)
fi

if [[ ${#FILES[@]} -eq 0 ]]; then
  echo "Error: No .textproto files found to test." >&2
  exit 1
fi

echo "======================================================="
echo "CEL Conformance Test Evaluator"
echo "======================================================="
if [[ -n "${TESTDATA_DIR}" ]]; then
  echo "Testdata Dir: ${TESTDATA_DIR}"
fi
echo "Files Found:  ${#FILES[@]}"
if [[ -n "${FILTER}" ]]; then
  echo "Filter:       ${FILTER}"
fi
echo "======================================================="
echo ""

TOTAL_FILES=0
TOTAL_TESTS=0
TOTAL_PASSED=0
TOTAL_SKIPPED=0
TOTAL_FAILED=0
FAILED_FILES=()

for file in "${FILES[@]}"; do
  filename="$(basename "$file")"

  # Skip network_ext by default if testing all files (requires non-core IP extension)
  if [[ -z "${SPECIFIC_FILE}" && "${filename}" == "network_ext.textproto" ]]; then
    printf "%-35s %s\n" "${filename}:" "SKIPPED (requires non-core network extensions)"
    continue
  fi

  TOTAL_FILES=$((TOTAL_FILES + 1))

  CMD=("${CLI_BIN}" "conformance" "-tests" "${file}" "-skip" "${ALL_SKIPS}")
  if [[ -n "${FILTER}" ]]; then
    CMD+=("-filter" "${FILTER}")
  fi

  JSON_OUTPUT=""
  STDERR_OUTPUT=""
  EXIT_CODE=0
  STDERR_FILE="$(mktemp)"
  if JSON_OUTPUT="$("${CMD[@]}" 2>"${STDERR_FILE}")"; then
    EXIT_CODE=0
  else
    EXIT_CODE=$?
  fi
  STDERR_OUTPUT="$(cat "${STDERR_FILE}")"
  rm -f "${STDERR_FILE}"

  if [[ "${VERBOSE}" == true ]]; then
    echo "--- ${filename} ---"
    echo "${JSON_OUTPUT}"
    if [[ -n "${STDERR_OUTPUT}" ]]; then
      echo "${STDERR_OUTPUT}" >&2
    fi
  fi

  # Parse JSON output using jq or python3
  FILE_TOTAL=0
  FILE_PASSED=0
  FILE_SKIPPED=0
  FILE_FAILED=0

  if command -v jq >/dev/null 2>&1; then
    FILE_TOTAL=$(echo "${JSON_OUTPUT}" | jq -r '.total // 0' 2>/dev/null || echo "0")
    FILE_PASSED=$(echo "${JSON_OUTPUT}" | jq -r '.passed // 0' 2>/dev/null || echo "0")
    FILE_SKIPPED=$(echo "${JSON_OUTPUT}" | jq -r '.skipped // 0' 2>/dev/null || echo "0")
    FILE_FAILED=$(echo "${JSON_OUTPUT}" | jq -r '.failed // 0' 2>/dev/null || echo "0")
  elif command -v python3 >/dev/null 2>&1; then
    read -r FILE_TOTAL FILE_PASSED FILE_SKIPPED FILE_FAILED < <(
      echo "${JSON_OUTPUT}" | python3 -c '
import sys, json
try:
    d = json.load(sys.stdin)
    print(d.get("total", 0), d.get("passed", 0), d.get("skipped", 0), d.get("failed", 0))
except Exception:
    print("0 0 0 0")
'
    )
  fi

  # Ensure numeric values
  [[ "${FILE_TOTAL}" =~ ^[0-9]+$ ]] || FILE_TOTAL=0
  [[ "${FILE_PASSED}" =~ ^[0-9]+$ ]] || FILE_PASSED=0
  [[ "${FILE_SKIPPED}" =~ ^[0-9]+$ ]] || FILE_SKIPPED=0
  [[ "${FILE_FAILED}" =~ ^[0-9]+$ ]] || FILE_FAILED=0

  TOTAL_TESTS=$((TOTAL_TESTS + FILE_TOTAL))
  TOTAL_PASSED=$((TOTAL_PASSED + FILE_PASSED))
  TOTAL_SKIPPED=$((TOTAL_SKIPPED + FILE_SKIPPED))
  TOTAL_FAILED=$((TOTAL_FAILED + FILE_FAILED))

  if [[ ${FILE_FAILED} -gt 0 || ${EXIT_CODE} -ne 0 ]]; then
    FAILED_FILES+=("${filename}")
    printf "%-35s \033[31mFAIL\033[0m (%d total, %d passed, %d skipped, %d failed)\n" \
      "${filename}:" "${FILE_TOTAL}" "${FILE_PASSED}" "${FILE_SKIPPED}" "${FILE_FAILED}"
    
    # Print failure details
    if command -v jq >/dev/null 2>&1; then
      echo "${JSON_OUTPUT}" | jq -r '.results[]? | select(.status != "pass" and .status != "skipped") | "    -> \(.name): \(.status)"' 2>/dev/null || true
    elif command -v python3 >/dev/null 2>&1; then
      echo "${JSON_OUTPUT}" | python3 -c '
import sys, json
try:
    d = json.load(sys.stdin)
    for r in d.get("results", []):
        if r.get("status") not in ("pass", "skipped"):
            print(f"    -> {r.get(\"name\")}: {r.get(\"status\")}")
except Exception:
    pass
' 2>/dev/null || true
    fi
    if [[ -n "${STDERR_OUTPUT}" && ${FILE_TOTAL} -eq 0 ]]; then
      echo "    -> ${STDERR_OUTPUT}" >&2
    fi
  else
    printf "%-35s \033[32mPASS\033[0m (%d total, %d passed, %d skipped, %d failed)\n" \
      "${filename}:" "${FILE_TOTAL}" "${FILE_PASSED}" "${FILE_SKIPPED}" "${FILE_FAILED}"
  fi
done

echo ""
echo "======================================================="
echo "CEL CONFORMANCE SUMMARY"
echo "======================================================="
echo "Files Evaluated: ${TOTAL_FILES}"
echo "Total Tests:     ${TOTAL_TESTS}"
echo "Passed:          ${TOTAL_PASSED}"
echo "Skipped:         ${TOTAL_SKIPPED}"
echo "Failed:          ${TOTAL_FAILED}"
echo "======================================================="

if [[ ${TOTAL_FAILED} -gt 0 || ${#FAILED_FILES[@]} -gt 0 ]]; then
  echo ""
  echo "FAILED SUITES: ${FAILED_FILES[*]}"
  exit 1
else
  echo ""
  echo "ALL CONFORMANCE TESTS PASSED!"
  exit 0
fi
