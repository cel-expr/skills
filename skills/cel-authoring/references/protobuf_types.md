# Protobuf Types in `cel-expr`

Custom protobuf messages and enums require descriptors via `-proto_db` or `-fds`
(CLI flags on `cel-expr`, or startup flags on `cel-expr mcp` / `cel-expr-mcp`).

## Loading Descriptors (`-proto_db` and `-fds`)

*   **`-proto_db`** (off by default in standard builds): Resolves protobuf types
    in `<env.json>` (`contextVariable`, `variables`, `functions`, `imports`, and
    `protoTypes`) from the binary's descriptor database. Always pass `-proto_db`
    explicitly—without it, `"protoTypes"` is ignored.
*   **`-fds <path>`** (`-file_descriptor_set`): Loads a binary
    `FileDescriptorSet` from `protoc`. Required for local `.proto` files not
    registered in `-proto_db` (standard Go builds only register linked-in
    types). May be combined with `-proto_db`:

    ```bash
    protoc --include_imports --descriptor_set_out=desc.fds \
      -I <proto_dir> <schema.proto>
    cel-expr eval -env <env.json> -fds desc.fds -expr "<expr>"
    ```

## Environment Declarations (`<env.json>`)

*   **Fully-qualified names only**: Types in `variables`, `contextVariable`,
    `functions`, and `protoTypes` must include their full package (e.g.,
    `acme.v1.Request`, `acme.v1.Request.Sub`). `container` and `imports`
    abbreviations apply only inside expressions.
*   **Enums are `int` in CEL**: Declare enum variables and function args/returns
    as `"int"`, never as the protobuf enum name. Write qualified constants
    (`acme.v1.Verdict.ALLOW`) in expressions, and use integers for `"int"` enum
    values in `-bindings` / `-tests` (enum fields inside message bindings accept
    `"ALLOW"` or `1` via `protojson`).
*   **Use `"protoTypes"`, never `"imports"`, to load descriptors**:
    `"protoTypes"` loads descriptors without altering CEL name resolution;
    `"imports"` registers short-name aliases that risk collisions or drift from
    production. `cel-expr convert-env` outputs start without `"protoTypes"`, and
    `-proto_db` does not scan `.celpolicy` `imports:`—copy policy-imported type
    names into `<env.json>` `"protoTypes"`.

## Diagnosing Protobuf Errors

*   **`undefined type name: "<T>"` or `invalid context proto type: "<T>"`**
    (environment error): `<T>` in `variables`, `contextVariable`, or `functions`
    is (1) a protobuf **enum** type (change to `"int"`; enums fail here even
    with descriptors loaded), (2) missing its package prefix, or (3) missing
    `-proto_db` / `-fds`.
*   **`proto_db: failed to resolve type "<T>"`** (stderr warning / error
    suffix): `<T>` is misspelled, wrongly prefixed, or defined in a local
    `.proto` file not in `-proto_db` (compile with `protoc` and `-fds`).
*   **`undeclared reference to '<id>'`** (compile error): Missing `variables` /
    `functions` entry, or an expression-only message, enum, or type literal
    missing from `"protoTypes"` (or `-proto_db` omitted).

## Inferring `"protoTypes"` from `undeclared reference`

Failed `Select` chains (enums, type literals, and undeclared namespaced
variables) quote **only the leftmost identifier** (`'acme'`), whereas struct
literals quote the written message path (`'acme.v1.Controls'`). Always inspect
the expression at `<line>:<col>`:

| Expression at `<line>:<col>` | Quoted error | Candidate symbol |
| :--- | :--- | :--- |
| `acme.v1.Controls{ok: true}` | `'acme.v1.Controls'` | `acme.v1.Controls` |
| `acme.v1.Msg.Sub{x: 1}` | `'acme.v1.Msg.Sub'` | `acme.v1.Msg.Sub` |
| `acme.v1.Verdict.ALLOW` | `'acme'` | `acme.v1.Verdict` |
| `acme.v1.Reason.Enum.OK` | `'acme'` | `acme.v1.Reason.Enum` |
| `type(x) == acme.v1.Request` | `'acme'` | `acme.v1.Request` |
| `Verdict.ALLOW` (short name) | `'Verdict'` | `Verdict` (qualify) |
| `request.path` / `ns.fn(x)` | `'request'` / `'ns'` | *(Fix vars/funcs)* |

1.  **Extract the candidate type**: Keep the full type path for `T{...}` and
    `type(x) == T`; strip the single trailing constant segment (`ALLOW`, `OK`,
    or proto2 `Allow`) from enum values `E.VALUE` (for nested enums, either
    `acme.v1.Reason.Enum` or `acme.v1.Reason` loads the `.proto`).
2.  **Qualify relative names**:
    *   **Leading dot** (`.acme.v1.Verdict`): Strip `.`; skip `container`.
    *   **Import alias**: If the first segment matches the final segment of an
        entry in `<env.json>` `"imports"` or `.celpolicy` `imports:` (e.g.,
        `Msg` -> `acme.v1.Msg`), expand it (`Msg.Sub` -> `acme.v1.Msg.Sub`).
    *   **Container**: Otherwise, if `"container"` is set (e.g.,
        `"acme.auth.v1"`) and `<T>` is relative, try `acme.auth.v1.<T>`, then
        parent prefixes `acme.auth.<T>`, `acme.<T>`, `<T>`.
3.  **Add to `"protoTypes"` and re-run with `-proto_db`**:

    ```json
    {
      "protoTypes": [
        "acme.v1.Controls",
        "acme.v1.Verdict",
        "acme.v1.Reason.Enum"
      ],
      "variables": [
        {"name": "req", "type": "acme.v1.Request"}
      ]
    }
    ```
