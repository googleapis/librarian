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
- [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments): Common
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

## Writing Tests

When writing tests, we follow the patterns below to ensure consistency,
readability, and ease of debugging. See
[Go Test Comments](https://go.dev/wiki/TestComments) for conventions around
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

### Use `cmp.Diff` for comparisons

Use [`go-cmp`](https://pkg.go.dev/github.com/google/go-cmp/cmp) instead of
`reflect.DeepEqual` for clearer diffs and better debugging.

Always compare in `want, got` order, and use this exact format for the error
message:

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

- Use `t.Run(test.name, ...)` to create subtests. Subtests can be run
  individually and parallelized when needed.

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
      name: "recipient does not exist",
      recipient: "Does Not Exist",
      message: "Hello, Mr. Not Exist",
      wantErr: errRecipientDoesNotExist,
    },
    {
      name: "empty message",
      recipient: "Jane Doe",
      message: "",
      wantErr: errEmptyMessage,
    },
  }{
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

## Need Help? Just Ask!

This guide will continue to evolve. If something feels unclear or is missing,
just ask. Our goal is to make writing Go approachable, consistent, and fun, so
we can build a high-quality, maintainable, and awesome Librarian CLI and system
together!
