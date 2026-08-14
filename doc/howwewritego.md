# How We Write Go

There are plenty of resources on the internet for how to write Go code. This
guide is about applying those rules to the Librarian codebase.

It covers the most important tools, patterns, and conventions to help you write
readable, idiomatic, and testable Go code in every pull request.

## Writing Effective Go

One of the core philosophies of Go is that
[clear is better than clever](https://www.youtube.com/watch?v=PAAkCSZUG1c&t=875s&ab_channel=TheGoProgrammingLanguage),
a principle captured in
[Go Proverbs](https://go-proverbs.github.io/).

While
[simplicity is complicated](https://go.dev/talks/2015/simplicity-is-complicated.slide#1),
writing simple, readable Go can easily be achievable by following the
conventions the community has already established.

For guidance, refer to the following resources:

- [Effective Go](https://go.dev/doc/effective_go): The canonical guide to
  writing idiomatic Go code.
- [Go Code Review Comments](styleguide/go-code-review-comments.md): Common
  feedback and best practices used in Go code reviews.
- [Google's Go Style Guide](https://google.github.io/styleguide/go/decisions):
  Google’s guidance on Go style and design decisions.
- [Idiomatic Go](https://dmitri.shuralyov.com/idiomatic-go): Common rules and
  conventions for writing idiomatic Go.

## Naming and Spelling

### Capitalization

For brands or words with more than 1 capital letter, lowercase all letters when
unexported. See
[details](https://dmitri.shuralyov.com/idiomatic-go#for-brands-or-words-with-more-than-1-capital-letter-lowercase-all-letters)
- **Good**: `oauthToken`, `githubClient`
- **Bad**: `oAuthToken`, `gitHubClient`

### Comments

Comments for humans always have a single space after the slashes. See
[details](https://dmitri.shuralyov.com/idiomatic-go#comments-for-humans-always-have-a-single-space-after-the-slashes)
- **Good**: `// This is a comment.`
- **Bad**: `//This is a comment.`

### Explain "why", not just "what"

Comments should explain the rationale behind non-obvious decisions, workarounds,
or hacks.
- If implementing a workaround for an external tool bug, include a link to the
  issue tracker.
- Use comments to explain nuanced generic helpers or complex type manipulations
  (e.g., why a double pointer is used).

Example:
```go
// We use a pointer generic here so we can reset the variable in-place
// to release memory early. See b/12345678.
func reset[T any](ptr *T) { ... }
```

### Keep implementation details out of doc comments

Doc comments (comments preceding exported or unexported package symbols) should
explain *what* the symbol does and its contract. Technical
implementation details,
historical context, or build-system-specific notes belong as standard comments
*inside* the function body, not in the doc comment.

Example:
```go
// FormatRuby formats the Ruby code using the standard formatter.
func FormatRuby(code string) (string, error) {
	// We need to escape backslashes here because the underlying rubocop
	// tool interprets them differently in this context.
	...
}
```

### Collection Names

Use singular form for collection repo/folder name. See
[details](https://dmitri.shuralyov.com/idiomatic-go#use-singular-form-for-collection-repo-folder-name)
- **Good**: `example/`, `image/`, `player/`
- **Bad**: `examples/`, `images/`, `players/`

### Consistent Spelling

Use consistent spelling of certain words, following
https://go.dev/wiki/Spelling. See
[details](https://dmitri.shuralyov.com/idiomatic-go#use-consistent-spelling-of-certain-words).
- **Good**: `unmarshaling`, `marshaling`, `canceled`
- **Bad**: `unmarshalling`, `marshalling`, `cancelled`

### Package Names

When naming packages, follow these two principles:

1.  **Avoid redundancy.** Go uses package names to provide context, so avoid repeating the package name within a type or function name.
    - **Good**: `git.ShowFile`, `client.New`
    - **Bad**: `git.GitShowFile`, `client.NewClient`

2.  **Describe the purpose.** Good package names are short and descriptive. Avoid generic names.
    - **Good:** `command`, `fetch`
    - **Bad:** `common`, `helper`, `util`

See [details](https://go.dev/doc/effective_go#package-names).

## Go Doc Comments

"Doc comments" are comments that appear immediately before top-level package,
const, func, type, and var declarations with no intervening newlines. Every
exported (capitalized) name should have a doc comment.

See [Go Doc Comments](https://go.dev/doc/comment) for details.

These comments are parsed by tools like
[go doc](https://pkg.go.dev/cmd/go#hdr-Show_documentation_for_package_or_symbol),
[pkg.go.dev](https://pkg.go.dev/),
and IDEs via
[gopls](https://pkg.go.dev/golang.org/x/tools/gopls). You can also view local
or private module docs using
[pkgsite](https://pkg.go.dev/golang.org/x/pkgsite/cmd/pkgsite).

## Writing Go

### Handling Errors

Go doesn’t use exceptions.
[Errors are returned as values](https://go.dev/blog/errors-are-values) and must
be explicitly checked.

For guidance on common patterns and anti-patterns, see the
[Go Wiki on Errors](https://go.dev/wiki/Errors).

When working with generics, refer to these resources for idiomatic error
handling:
- [Generics Tutorial](https://go.dev/doc/tutorial/generics)
- [Error Handling with Generics](https://go.dev/blog/error-syntax)

Prefer `errors.Is(err, fs.ErrNotExist)` over `os.IsNotExist(err)` as described
in the [documentation for `os.IsNotExist`](https://pkg.go.dev/os#IsNotExist),
and prefer `fs.ErrNotExist` over `os.ErrNotExist` in general.

### Export sentinel errors

If a package exposes public functions that return sentinel errors, those errors
must be exported (capitalized) and documented. This allows callers to
programmatically inspect the error type using `errors.Is`.

- **Good**:
  ```go
  // In serviceconfig package:

  // ErrReadServiceConfig indicates that the service configuration could not be read.
  var ErrReadServiceConfig = errors.New("failed to read service config")

  func Read(...) error { ... return ErrReadServiceConfig }
  ```
- **Bad**:
  ```go
  // In serviceconfig package:

  // bad: sentinel error is unexported, preventing callers from using errors.Is
  var errReadServiceConfig = errors.New("failed to read service config")
  ```

### Wrapping errors

Whether to wrap an error or not is discussed at length in the
[Go Blog post](https://go.dev/blog/go1.13-errors#whether-to-wrap). Best practice
for _how_ to wrap errors is described in the
[Go Style Guide](https://google.github.io/styleguide/go/best-practices#placement-of-w-in-errors)
and summarized below.

When using the `%w` format specifier via `fmt.Errorf`, prefer to put it at the
end of the string, prepending the added context of where the error is being
handled, creating a chain of errors. For example:

```go
f, err := os.Open("special.cfg")
if err != nil {
  return fmt.Errorf("failed opening special config: %w", err)
}
```

However, when including a sentinel error in the newly formatted error, the `%w`
specifier for that sentinel error should appear at the front of the formatted
error, as per the [Go Style Guide](https://google.github.io/styleguide/go/best-practices#sentinel-error-placement).

### Avoid redundant error wrapping

Only wrap errors when adding useful operational context. If a helper function
already returns a highly descriptive error that explains the failure and
resolution, do not wrap it with a generic message that only repeats what the
function was doing.

- **Good**:
  ```go
  if err := validateModel(model); err != nil {
  	return err // validateModel already returns detailed validation error
  }
  ```
- **Bad**:
  ```go
  if err := validateModel(model); err != nil {
  	return fmt.Errorf("failed to validate model: %w", err)
  }
  ```

### Avoid unnecessary `else`

To keep the main logic flow linear and reduce indentation, return early or
continue early instead of using `else` blocks.

```go
// Good
if err != nil {
    return err
}
// process success case

// Bad
if err == nil {
    // process success case
} else {
    return err
}
```

Similarly, in a loop, use `continue` to skip to the next iteration instead of
wrapping the main logic in an `else` block.

```go
// Good
for _, item := range items {
    if item.skip {
        continue
    }
    // process item
}

// Bad
for _, item := range items {
    if !item.skip {
        // process item
    }
}
```

### Make mutations explicit

When a function modifies a pointer parameter, return the modified value to make
the mutation explicit. This makes it so that functions are clear about their
side effects.

```go
// Good: Returns the modified value to signal mutation
func UpdateConfig(cfg *Config) (*Config, error) {
    // ... update fields ...
    cfg.Version = newVersion
    return cfg, nil
}

// Usage makes mutation visible
cfg, err := UpdateConfig(config)

// Bad: Mutation is hidden
func UpdateConfig(cfg *Config) error {
    // ... update fields ...
    cfg.Version = newVersion
    return nil
}

// Usage hides that config was modified
err := UpdateConfig(config)
```

This pattern helps readers understand at a glance which functions modify their
inputs versus which functions only read them.

### Avoid magic numbers and hardcoded paths

Do not embed unexplained numeric literals (such as arbitrary limits or buffer
sizes) or hardcoded API and filesystem paths in business logic. Define them as
named constants with a comment explaining why that value was chosen, or pass
them via configuration.

```go
// Good
const maxStagingLines = 50 // Keep generated staging headers within line limit.

// Bad
if len(lines) > 50 {
    // ...
}
```

### Group declarations

When declaring multiple package-level or helper variables and constants, group
them into a single block rather than writing separate consecutive declarations.

```go
// Good
var (
    errInvalidPath = errors.New("invalid path")
    errMissingTool = errors.New("missing tool")
)

// Bad
var errInvalidPath = errors.New("invalid path")
var errMissingTool = errors.New("missing tool")
```

### Deterministic map iteration

Map iteration order in Go is non-deterministic. When iterating over a map to
generate code, produce user-facing output, or aggregate error lists, sort the
map keys first to guarantee deterministic behavior.

```go
// Good
keys := make([]string, 0, len(modules))
for name := range modules {
    keys = append(keys, name)
}
sort.Strings(keys)
for _, name := range keys {
    // Process modules in deterministic order
}
```

### Raw string literals and escape sequences

Raw string literals enclosed in backticks (`` ` ``) do not interpret escape
sequences such as `\n` or `\t`. When string output requires interpreted escape
characters, use double-quoted interpreted string literals or literal newlines
in raw strings.

### File length and organization

When a file grows large or encompasses multiple distinct sub-concerns, split
the logic into cohesive sub-files within the same package (for example,
separating `generate_post_hybrid.go` from `generate.go`). Keep files focused on
a well-defined scope.

### Prefer standard library slices and maps

Avoid writing manual loops to search, filter, clone, or compare slices and maps.
Use the built-in functions from the standard library `slices` and `maps`
packages (introduced in Go 1.21).

- **Good**:
  ```go
  hasConvertedFields := slices.ContainsFunc(message.Fields, func(f *api.Field) bool {
  	return !f.IsOneOf && f.Singular()
  })
  ```
- **Bad**:
  ```go
  hasConvertedFields := false
  for _, f := range message.Fields {
  	if !f.IsOneOf && f.Singular() {
  		hasConvertedFields = true
  		break
  	}
  }
  ```

### Avoid variable shadowing

Be careful when using the short variable declaration operator `:=` in nested
blocks (like `if`, `for`, or `switch` blocks). If you intend to update a
variable defined in an outer scope (like a parameter or a return value), use
assignment `=` instead of `:=`.

- **Good**:
  ```go
  var client *secretmanager.Client
  if client == nil {
  	var err error
  	client, err = secretmanager.NewClient(ctx) // Updates the outer 'client'
  	if err != nil {
  		return err
  	}
  }
  ```
- **Bad**:
  ```go
  var client *secretmanager.Client
  if client == nil {
  	client, err := secretmanager.NewClient(ctx) // Shadowing! Creates a new 'client' local to this block
  	if err != nil {
  		return err
  	}
  	log.Printf("Initialized client: %v", client) // Compiler is happy with local usage
  }
  // Outer 'client' is still nil here, leading to a nil pointer panic
  client.DoSomething()
  ```

### Clean function signatures

- **Remove Unused Parameters**: If a parameter is no longer used within the
function body, remove it from the signature and update all call sites.
- **Avoid Over-Parameterizing**: Do not pass configuration options to internal
helpers if they are constant across all current call sites. Use package or
function-level constants instead.

### Reuse and extend tool abstractions

Prefer extending existing tool packages (e.g., `composer`, `pnpm`, `pip`) to
support new use cases (such as local installations) rather than writing ad-hoc
`exec.Command` calls in business logic. This ensures consistent execution
environments, logging, and error handling.

### Move generated code strings out of Go logic

When generating code for other languages (e.g., Rust, Python, Swift), avoid
embedding large blocks of foreign code directly as formatted inline strings.
Move these code templates into package-level constants or separate template
files (e.g., mustache templates) to keep the Go logic clean.

## Writing Tests

When writing tests, we follow the patterns below to ensure consistency,
readability, and ease of debugging. See
[Go Test Comments](styleguide/go-test-comments.md) for conventions around
writing test code.

### Use `t.Context()`

Always use `t.Context()` instead of `context.Background()` in tests to ensure
proper cancellation and cleanup.

Example:
```go
err := Run(t.Context(), []string{"cmd", "arg"})
```

### Use `t.TempDir()`

Always use `t.TempDir()` instead of manually creating and cleaning up temporary
directories.

Example:
```go
err := Run(t.Context(), []string{"cmd", "-output", t.TempDir()})
```

### Use `t.Fatal` or `t.Error` for simple error handling

Avoid verbose or redundant failure messages. If an error occurs, pass it directly
to `t.Fatal` or `t.Error`. The testing package automatically includes the file
and line number, and well-constructed errors already provide their own context.

**Good**:
```go
t.Fatal(err)
```

**Bad**:
```go
t.Fatalf("failed: %v", err)
```

Only use `t.Fatalf` if you need to provide extra context not present in the
error, such as:
```go
t.Fatalf("failed to process user %d: %v", userID, err)
```

### Use `cmp.Diff` for comparisons

Use [`go-cmp`](https://pkg.go.dev/github.com/google/go-cmp/cmp) instead of
`reflect.DeepEqual` for clearer diffs and better debugging.

Always compare in `want, got` order, and use this exact format for the error
message (do not modify or customize this format string):

```go
t.Errorf("mismatch (-want +got):\n%s", diff)
```

Example:

```go
func TestGreet(t *testing.T) {
	got := Greet("Alice")
	want := "Hello, Alice!"

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
```

This format makes test failures easier to scan, especially when comparing
multiline strings or nested structs.

### Table-driven tests

Use table-driven tests to keep test cases compact, extensible, and easy to
scan. They make it straightforward to add new scenarios and reduce repetition.

Use this structure:

- Write `for _, test := range []struct { ... }{ ... }` directly. Don't name the
  slice. This makes the code more concise and easier to grep.

- **Loop Variable**: Always use `test` as the loop variable name
  (avoid abbreviations like `tt` or `tc`).

- Use `t.Run(test.name, ...)` to create subtests. Subtests can be run
  individually and parallelized when needed.

- **Simplify Test Case Structs**: Avoid adding boolean flags to the test case
  struct if the same logic can be controlled by checking the zero-value or empty
  status of an existing field (e.g., checking `wantErr != nil` or
  `content != ""`).

- Do not use table-driven tests when there is only a single test case. Write a
  straightforward unit test without the table boilerplate until more test cases
  are added.

Example:

```go
func TestTransform(t *testing.T) {
	for _, test := range []struct {
		name  string
		input string
		want  string
	}{
		{"uppercase", "hello", "HELLO"},
		{"empty", "", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := Transform(test.input)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
```

### Verify with `errors.Is` and `errors.AsType`

Avoid comparing error strings directly with `==` or `strings.Contains`, as these
are fragile and can break if the error message is updated for humans.

Instead:

- Use `errors.Is(err, target)` to check if an error matches a specific sentinel
  error (e.g., `fs.ErrNotExist`). It correctly handles wrapped errors.
- Use `errors.AsType[E](err)` when you need to verify if an error is of a
  specific custom type and access its fields (e.g., a `ValidationError` struct
	with a list of invalid fields).

### Separate error tests

Splitting success and failure cases into separate test functions can simplify
your test code. See
[details](https://google.github.io/styleguide/go/decisions.html#table-driven-tests).

When writing error tests, use a test function name like `TestXxx_Error`, and
when possible use [`errors.Is`](https://pkg.go.dev/errors#Is) for comparison
(see [details](https://google.github.io/styleguide/go/decisions.html#test-error-semantics)).

Example:

```go
func TestSendMessage_Error(t *testing.T) {
	for _, test := range []struct {
		name      string
		recipient string
		message   string
		wantErr   error
	}{
		{
			name:      "recipient does not exist",
			recipient: "Does Not Exist",
			message:   "Hello, Mr. Not Exist",
			wantErr:   errRecipientDoesNotExist,
		},
		{
			name:      "empty message",
			recipient: "Jane Doe",
			message:   "",
			wantErr:   errEmptyMessage,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, gotErr := SendMessage(test.recipient, test.message)
			if !errors.Is(gotErr, test.wantErr) {
				t.Errorf("SendMessage(%q, %q) error = %v, wantErr %v", test.recipient, test.message, gotErr, test.wantErr)
			}
		})
	}
}
```

### Running tests in parallel

Large table-driven tests and/or those that are I/O bound e.g. by making
filesystem reads or network requests are good candidates for parallelization via
[`t.Parallel()`](https://pkg.go.dev/testing#T.Parallel). Do not
parallelize lightweight, millisecond-level tests.

**Important:** A test cannot be parallelized if it depends on shared resources,
mutates the process as a whole e.g. by invoking `t.Chdir()`, or is dependent on
execution order.

```go
func TestTransform(t *testing.T) {
	for _, test := range []struct {
		name  string
		input string
		want  string
	}{
		{"uppercase", "hello", "HELLO"},
		{"empty", "", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel() // Mark subtest for parallel execution.
			got := Transform(test.input)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
```

### Readability of test expectations

Use raw string literals (backticks `` ` ``) when defining multi-line string
expectations in tests (such as expected generated code, templates, or
JSON payloads).
This avoids the need for escape sequences (like `\n` or `\"`) and keeps the
expected output readable.

### Hardcode expected values

When defining expected outputs (`want` values) in tests, write them as static,
hardcoded literals rather than computing them using production helper functions.
This ensures that bugs in the helpers don't mask bugs in the code being tested.

### Keep tests hermetic

Tests must not rely on files or directory structures outside of their own
package or a `testdata/` subdirectory. If a test requires mock files or
configurations, place them in `testdata/` or set them up dynamically in the
test code.

### Test file layout

In test files, place test functions (`TestXxx`) at the top of the file and
helper functions (e.g., custom assert helpers, setup/teardown functions)
at the bottom.

### Reuse existing test helpers

Before creating new test helpers, custom mocks, or manually initializing
complex structs, check `internal/testhelper` and existing package utilities
(such as `api.*` test constructors in `internal/sidekick/api/test.go`).
Reusing shared helpers keeps test data consistent and reduces duplication.

## Logging

We use Go's standard library `log/slog` for structured logging. The CLI
configures a global default text handler that outputs to `os.Stderr`.

### Log Levels

- **Default**: Logs are filtered to `slog.LevelWarn` and above.
- **Verbose**: If the verbose flag is set (e.g., `librarian -v` or `librarian --verbose`), the level is set to `slog.LevelDebug`, displaying all logs.

### Best Practices

1. **Prefer returning errors**: Deeply nested functions should prefer returning errors rather than logging and returning nil/empty values.
2. **Log at boundaries**: Logging should be reserved for cross-cutting execution concerns, warnings where execution can proceed, or debugging info.
3. **Provide structured context**: Always pass structured key-value pairs to log messages.
4. **Maintain consistent keys**: When using structured logging keys (e.g., `"api"`), use them consistently across log sites.

   ```go
   slog.Warn("missing version, defaulting to 0.1.0", "api", api, "error", err)
   ```

## Package-Specific Rules

### `internal/config`

The `internal/config` package is a direct 1:1 mapping with `librarian.yaml`. It
contains only struct types and constants that mirror the YAML schema. Do not add
functions or methods to this package. Any logic that operates on configuration
values belongs in the package that uses them, not in `internal/config` itself.

## Need Help? Just Ask!

This guide will continue to evolve. If something feels unclear or is missing,
just ask. Our goal is to make writing Go approachable, consistent, and fun, so
we can build a high-quality, maintainable, and awesome Librarian CLI and system
together!
