---
name: cel-optimize
description: >-
  Optimizes, simplifies, and refactors Common Expression Language (CEL)
  expressions and policies while formally proving semantic equivalence using
  the CEL-Java Verifier. Use when simplifying complex or redundant CEL
  expressions, eliminating duplicate subexpressions with cel.bind,
  consolidating boolean or membership logic, modernizing presence checks to
  optional chaining, or verifying that two CEL expressions or policies are
  logically equivalent across all inputs. Don't use for authoring new CEL
  expressions from scratch (use cel-authoring), fixing broken expressions that
  fail to compile (use cel-debugging), or generating unit test suites without
  refactoring (use cel-testing).
---

# Common Expression Language (CEL) Optimization & Verification Skill

Use this skill to simplify Common Expression Language (CEL) expressions and
formally prove semantic equivalence across all inputs using `cel-verifier` and
`cel-expr`:

```bash
# Download the cel-java verifier-cli JAR from Maven Central (requires Java 8+)
curl -fsSL --create-dirs -o "$HOME/.local/lib/cel/verifier-cli-0.14.0.jar" https://repo1.maven.org/maven2/dev/cel/verifier-cli/0.14.0/verifier-cli-0.14.0.jar
alias cel-verifier="java -jar $HOME/.local/lib/cel/verifier-cli-0.14.0.jar"
go install github.com/cel-expr/skills/cmd/cel-expr@latest
alias cel-expr="$(go env GOPATH)/bin/cel-expr"
```

> **Note**: In subshell-per-command environments, define aliases in the same
> command block or invoke binary paths directly. Run bare `cel-verifier` (no
> arguments) for top-level command usage.

## Workflow

1.  **Validate Baseline**: Inspect `<env.json>` (variables and enabled
    `extensions`) and compile the original expression with `cel-expr compile
    -env <env.json> -expr "<original_expr>"` (for standalone expressions using
    `?` without an `<env.json>` file, pass inline `-env
    '{"extensions":[{"name":"optional","version":"latest"}]}'`). Fix any
    existing errors first using
    [cel-debugging](skills/cel-debugging/SKILL.md).
2.  **Simplify**: Apply boolean, membership (`in`), constant folding,
    optional, or `cel.bind` simplifications. Only introduce `cel.bind`
    if `bindings` is enabled in `<env.json>`, and `.?` / `[?]` / `optional.*` if
    `optional` is enabled (`cel-verifier` enables standard extensions
    unconditionally, so check `<env.json>` first).
3.  **Compile Candidate**: Run `cel-expr compile -env <env.json> -expr
    "<optimized_expr>"` to verify types and target environment compatibility.
4.  **Prove Equivalence**: Run `cel-verifier verify-equiv` with `--expr1`,
    `--expr2`, and `--var "<name>:<type>"` for each variable.
5.  **Resolve Outcomes**:
    *   `[VERIFIED]` (exit `0`): Proven semantically equivalent.
    *   `[VIOLATED]` (exit `1`): Inspect the reported counterexample, fix the
        edge case (error preservation, `NaN`, missing keys), and re-verify.
    *   `[INCONCLUSIVE]` (exit `2`): Abstract uninterpreted calls into symbolic
        `--var` variables to verify surrounding logic, and test full expressions
        with `cel-expr eval` (>95% branch coverage; note `cel.bind`
        comprehension macros leave synthetic AST nodes unvisited in node
        coverage) via
        [cel-testing](skills/cel-testing/SKILL.md).

## 1. Formal Verification (`cel-verifier`)

`cel-verifier` translates CEL ASTs into Z3 SMT formulas under CEL's 4-valued
logic (`True`, `False`, `Error`, `Unknown`):

```bash
cel-verifier verify-equiv \
  --expr1 "<original_expr>" \
  --expr2 "<optimized_expr>" \
  --var "request.path:string" \
  --var "request.headers:map<string, string>" \
  --var "user.age:int"
```

*   **Supported `--var` (`-v`) Types**: `int`, `uint`, `string`, `bool`,
    `double`, `bytes`, `dyn`, `timestamp`, `duration`, `list<T>`, `map<K, V>`,
    and `optional<T>`. Namespaced variables (e.g., `request.path`) are
    supported.
*   **Other Subcommands**:
    *   `check-sat --expr "<expr>"`: Dead branch check (`[VIOLATED]` ⇒
        unsatisfiable, simplify to `false`).
    *   `check-valid --expr "<expr>"`: Tautology check (`[VERIFIED]` ⇒ always
        true, simplify to `true`).
    *   `verify-policy --file <policy.yaml>`: Verify YAML policy invariants.
    *   `repl`: Batch multiple checks in a single JVM/Z3 session instead of
        spawning parallel processes:

        ```bash
        printf ':var x int\n:var opt_x optional<int>\nequiv x > 10 && x > 5 <=> x > 10\nequiv [1, ?optional.of(2), ?optional.none(), ?opt_x] <=> [1, 2, ?opt_x]\n:quit\n' | cel-verifier repl
        ```

*   **Flags**: `--unknown <id>` (`-u`), `--unroll-limit <int>` (BMC loop bound,
    default `5`), `--timeout <sec>` (default `10`),
    `--output_format <TEXT|JSON>` (`-fmt`).

## 2. Safe CEL Simplification Patterns

### Boolean, Relational, Membership & Constant Folding

| Pattern | Unoptimized (`--expr1`) | Simplified (`--expr2`) |
| :--- | :--- | :--- |
| Bound Subsumption | `age >= 18 && age > 12` | `age >= 18` |
| Absorption | `is_admin \|\| (is_admin && is_active)` | `is_admin` |
| Complement (`bool` var) | `(a && b) \|\| (a && !b)` | `a` |
| De Morgan (Boolean) | `!(a && b)` | `!a \|\| !b` |
| De Morgan (Quantifier) | `!items.exists(x, x <= 0)` | `items.all(x, x > 0)` |
| Membership (`in`) | `r == 'admin' \|\| r == 'editor'` | `r in ['admin', 'editor']` |
| Negated Membership | `r != 'guest' && r != 'banned'` | `!(r in ['guest', 'banned'])` |
| Constant Folding | `timeout > 60 * 60 && 'a' + 'b' == s` | `timeout > 3600 && 'ab' == s` |

### Common Subexpression Elimination (`cel.bind`)

Factor repeated traversals or transformations into `cel.bind(var, init, body)`
(requires `bindings` extension in `<env.json>`). Keep `cel.bind` *inside*
presence/key guards so `init` does not fail when the guard is `false`:

```cel
// Before:
'authorization' in request.headers &&
request.headers['authorization'].lowerAscii().startsWith('bearer ') &&
request.headers['authorization'].lowerAscii().size() > 20

// After:
'authorization' in request.headers &&
cel.bind(auth, request.headers['authorization'].lowerAscii(),
  auth.startsWith('bearer ') && auth.size() > 20
)
```

### Optional Value Equivalence & Literal Constructions (`optional<T>`)

`cel-verifier` natively proves `optional<T>` equivalence at the top level of
`verify-equiv`, via `==` / `!=`, and inside container literals (`list`, `map`,
`struct`):

*   **`optional.of(...)` & `optional.none()` Equivalence**:
    *   Scalars & folding: `optional.of(1 + 2)` ⇔ `optional.of(3)`,
        `optional.of('a' + 'b')` ⇔ `optional.of('ab')`
    *   Wrapped containers (via extensionality axioms):
        *   `optional.of([1, 1 + 1])` ⇔ `optional.of([1, 2])`
        *   `optional.of({'a': 1, 'b': 2})` ⇔ `optional.of({'b': 2, 'a': 1})`
        *   `optional.of([1, ?optional.of(2), ?optional.none()])` ⇔
            `optional.of([1, 2])`
        *   `optional.of({'a': 1, ?'b': optional.of(2), ?'c': optional.none()})`
            ⇔ `optional.of({'a': 1, 'b': 2})`
    *   Equality: `optional.of(x) == optional.of(y)` ⇔ `x == y` (for
        non-`double`, non-`dyn`), `optional.of(x) == optional.none()` ⇔ `false`,
        `optional.none() == optional.none()` ⇔ `true`.
        
*   **`optional.ofNonZeroValue(...)` Literal Simplifications**:
    *   **Zero/Empty Literals ⇔ `optional.none()`**: `0`, `0u`, `0.0`, `''`,
        `false`, `b''`, `timestamp(0)`, `duration('0s')`, `[]`,
        `[?optional.none()]`, `{}`, `{?'a': optional.none()}`, `null`, and empty
        Proto3 messages (`Msg{}`).
    *   **Non-Zero Literals ⇔ `optional.of(v)`**: `42`, `42u`, `3.14`, `'hi'`,
        `true`, `b'abc'`, `[1, 2]`, `{'k': 1}`, `Msg{field: 1}`, or folded
        expressions (`optional.ofNonZeroValue(1 + 2)` ⇔ `optional.of(3)`).
        
*   **Optional Element & Entry Pruning (`?`)**:
    *   Lists: `[1, ?optional.of(2), ?optional.none(), ?opt_x]` ⇔ `[1, 2,
        ?opt_x]`
    *   Maps: `{'a': 1, ?'b': optional.of(2), ?'c': optional.none()}` ⇔ `{'a':
        1, 'b': 2}`
    *   Messages: `Msg{?repeated_int32: optional.of([1, 2]),` `?single_string:
        optional.none()}` ⇔ `Msg{repeated_int32: [1, 2]}`
        
*   **Chaining, Unwrapping & Combinators**:
    *   Guarded ternaries: `'k' in m ? m['k'] : d` ⇔ `m[?'k'].orValue(d)`;
        `has(msg.f) ? msg.f : d` ⇔ `msg.?f.orValue(d)`
    *   Unwrapping: `optional.of(x).orValue(y)` ⇔ `x`;
        `optional.none().orValue(y)` ⇔ `y`; `optional.of(x).value()` ⇔ `x`;
        `optional.of(x).hasValue()` ⇔ `true`; `optional.none().hasValue()` ⇔
        `false`
    *   Fallback chains: `optional.of(x).or(opt_y)` ⇔ `optional.of(x)`;
        `optional.none().or(opt_y)` ⇔ `opt_y`; `opt_x.or(optional.none())` ⇔
        `opt_x`
    *   Literal indexing/selection: `[10, 20][?1]` / `optional.of([10, 20])[?1]`
        ⇔ `optional.of(20)`; `[10, 20][?5]` / `optional.none()[?0]` ⇔
        `optional.none()`; `{'a': {'b': 42}}.?a.?b` ⇔ `optional.of(42)`; `{'a':
        1}[?'missing']` / `{'a': 1}.?missing` ⇔ `optional.none()`

## 3. Semantic Pitfalls & `[INCONCLUSIVE]` Handling

*   **Error Preservation & Narrowing**: Removing a guard around a failing or
    narrowing operation is `[VIOLATED]` (`'k' in m && m['k'] == 'v'` ⇎ `m['k']
    == 'v'` when `m = {}`; `x / x == 1` ⇎ `true` when `x = 0`).
*   **`NaN` & `dyn` Reflexivity**: `d == d` and `v in [1, v]` are `[VIOLATED]`
    for `double` / `dyn` when `d = NaN`. Reflexive equality (`x == x` ⇔ `true`)
    is only valid for `int`, `uint`, `string`, `bool`, `bytes`.
*   **Proto3 Scalar Defaults**: `has(msg.single_int32) ? msg.single_int32 : 0`
    ⇔ `msg.single_int32` (`0` is default; fallback `1` is `[VIOLATED]`).
    `Msg{single_int32: 0}.?single_int32` ⇔ `optional.none()`.
*   **Symbolic Optionals & Null Map Values**: `[1, ?opt_var]` cannot be pruned
    to `[1]` or `[1, opt_var]`. On `map<string, dyn>`, `{'a': null}.?a` is
    `optional.of(null)` and `{'a': optional.none()}.?a` is
    `optional.of(optional.none())`, not `optional.none()`.
*   **Uninterpreted Functions, Loop Bounds & `[INCONCLUSIVE]`**: Functions
    without SMT axioms (`matches`, `lowerAscii`, `split`, `replace`, custom
    functions) use congruence closure—`cel-verifier` still returns `[VERIFIED]`
    when their calls are preserved identically (e.g., via `cel.bind`) or
    eliminated by boolean absorption. Comprehensions (`exists`, `all`, `map`,
    `filter`) over unbounded symbolic lists/maps return `[INCONCLUSIVE]` above
    `--unroll-limit` (default `5`) unless `size() <= <unroll_limit>` holds. If
    `[INCONCLUSIVE]` occurs, verify under a bounded `size()` guard or symbolic
    `--var` abstraction and validate with `cel-expr eval`.

## Complete Example

*   Environment:
    [auth_policy_env.json](skills/cel-optimize/examples/auth_policy_env.json)
*   Unoptimized:
    [auth_policy_unoptimized.cel](skills/cel-optimize/examples/auth_policy_unoptimized.cel)
*   Optimized:
    [auth_policy_optimized.cel](skills/cel-optimize/examples/auth_policy_optimized.cel)
*   Tests:
    [auth_policy_tests.json](skills/cel-optimize/examples/auth_policy_tests.json)

```bash
cel-verifier verify-equiv \
  --expr1 "$(cat skills/cel-optimize/examples/auth_policy_unoptimized.cel)" \
  --expr2 "$(cat skills/cel-optimize/examples/auth_policy_optimized.cel)" \
  --var "request.method:string" \
  --var "request.path:string" \
  --var "request.headers:map<string, string>" \
  --var "user.role:string" \
  --var "user.age:int"
```
