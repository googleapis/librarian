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

While [simplicity is complicated](https://go.dev/talks/2015/simplicity-is-complicated.slide#1),
writing simple, readable Go can easily be achievable by following the
conventions the community has already established.

For guidance, refer to the following resources:

- [Effective Go](https://go.dev/doc/effective_go): The canonical guide to
  writing idiomatic Go code.
- [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments): Common
  feedback and best practices used in Go code reviews.
- [Google's Go Style Guide](https://google.github.io/styleguide/go/decisions):
  Google’s guidance on Go style and design decisions.


## Writing Go Doc Comments

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


## Writing Tests

When writing tests, we follow the patterns below to ensure consistency,
readability, and ease of debugging. See
[Go Test Comments](https://go.dev/wiki/TestComments) for conventions around
writing test code.

### Use `cmp.Diff` for comparisons

Use [`go-cmp`](https://pkg.go.dev/github.com/google/go-cmp/cmp) instead of
`reflect.DeepEqual` for clearer diffs and better debugging.

Always compare in `want, got` order, and use this exact format for the error message:

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
				t.Errorf("Transform() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
```

## Handling Errors

Go doesn’t use exceptions.
[Errors are returned as values](https://go.dev/blog/errors-are-values) and must
be explicitly checked.

For guidance on common patterns and anti-patterns, see the
[Go Wiki on Errors](https://go.dev/wiki/Errors).

When working with generics, refer to these resources for idiomatic error
handling:
- [Generics Tutorial](https://go.dev/doc/tutorial/generics)
- [Error Handling with Generics](https://go.dev/blog/error-syntax)


## Need Help? Just Ask!

This guide will continue to evolve. If something feels unclear or is missing,
just ask. Our goal is to make writing Go approachable, consistent, and fun, so
we can build a high-quality, maintainable, and awesome Librarian CLI and system
together!
