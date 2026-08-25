## JWT

JSON Web Token (JWT) data types and helper functions for parsing tokens, claims inspection, and issuer/audience verification.

Returns a `cel.EnvOption` to configure support for JWT parsing and claims extraction. To use this extension, import `cel.dev/cel-go/ext/security/jwt` and pass `jwt.Library()` to `cel.NewEnv()`.

Note, the parser function uses the 'jwt' namespace. If you are currently using a variable named 'jwt', the function will likely work just as intended; however, there is some chance for collision.

### JWT.Parse

**Introduced in version 0**

Parses a raw token string into a structured `jwt.Token` representation wrapped in an `optional`. Automatically strips leading `Bearer ` prefixes if present. If the token string is malformed, or if time validation options are enabled and the token is expired/not yet valid, `optional.none()` or a parse error is returned.

```
jwt.parse(<string>) -> <optional(jwt.Token)>
```

Examples:

```
jwt.parse(tokenStr).hasValue()                          // true
jwt.parse('Bearer ' + tokenStr).hasValue()              // true
jwt.parse(tokenStr).value().alg == 'RS256'              // true
jwt.parse(tokenStr).value().issuer == 'https://auth.com' // true
jwt.parse('invalid.token')                              // error
```

### Claim

**Introduced in version 0**

Queries a claim value by key name from the JWT token payload, returning an optional dynamic value. This method can be called directly on a `jwt.Token` or chained directly on an `optional(jwt.Token)`.

```
<jwt.Token>.claim(<string>) -> <optional(dyn)>
<optional(jwt.Token)>.claim(<string>) -> <optional(dyn)>
```

Examples:

```
jwt.parse(tokenStr).value().claim('tenant').orValue('') // 'tenant_abc'
jwt.parse(tokenStr).claim('tenant').orValue('')         // 'tenant_abc'
jwt.parse(tokenStr).claim('nonexistent').hasValue()     // false
```

### PresentedBy

**Introduced in version 0**

Validates that the token was issued by the specified issuer and is targeted to the specified audience. Returns `true` if `token.issuer == expectedIssuer` and `expectedAudience in token.aud`. Can be invoked on a `jwt.Token` or chained directly on an `optional(jwt.Token)`.

```
<jwt.Token>.presentedBy(<string>, <string>) -> bool
<optional(jwt.Token)>.presentedBy(<string>, <string>) -> bool
```

Examples:

```
jwt.parse(tokenStr).presentedBy('https://auth.example.com', 'https://api.example.com') // true
jwt.parse(tokenStr).presentedBy('https://evil.com', 'https://api.example.com')        // false
jwt.parse(tokenStr).presentedBy('https://auth.example.com', 'https://wrong-aud.com')   // false
```

### jwt.Token

*   **Description**: Native object type representing a validated JSON Web Token.
*   **Fields**:
    *   `alg` (`string`): Cryptographic signing algorithm from header (e.g. `'RS256'`).
    *   `keyId` (`string`): Key ID (`kid`) from header.
    *   `issuer` (`string`): Token issuer claim (`iss`).
    *   `subject` (`string`): Subject claim (`sub`).
    *   `id` (`string`): Unique token identifier (`jti`).
    *   `aud` (`list(string)`): Target audience list (`aud`).
*   **Methods**:
    *   `<jwt.Token>.claim(<string>)` returns `<optional(dyn)>`.
    *   `<jwt.Token>.presentedBy(<string>, <string>)` returns `bool`.
