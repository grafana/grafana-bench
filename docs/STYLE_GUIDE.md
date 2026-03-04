# Documentation Style Guide

This guide ensures consistency across all Grafana Bench documentation.

## Markdown Formatting Standards

### Headers

- Use ATX-style headers (`# Header`)
- Title case for H1, sentence case for H2-H6
- One H1 per document (document title)
- Single blank line before and after headers

**Good:**
```markdown
# Main Title

## Section heading

Content here.

## Another section
```

**Bad:**
```markdown
## No H1 Title
### Skipped H2

Content without spacing.
##No spacing
```

### Code Blocks

- Always use fenced code blocks with language tags
- Language tags: `sh` for shell commands (Alpine-compatible), `yaml` (not yml), `typescript`, `go`, `json`
- Use `bash` only for bash-specific syntax (arrays, process substitution, etc.)
- Indent code block content properly (no leading spaces in fence)

**Good:**
````markdown
```sh
grafana-bench test \
  --suite-name my-repo/tests \
  --suite-path ./tests
```
````

**Bad:**
````markdown
```
    grafana-bench test \
      --suite-name my-repo/tests \
      --suite-path ./tests  # Indented
```
````

or

````markdown
```shell  # Use sh not shell
grafana-bench test \
  --suite-name my-repo/tests \
  --suite-path ./tests
```
````

**Rationale:** Bench uses Alpine Linux docker images where `sh` is available but `bash` is not installed by default. Use `sh` for portability unless you need bash-specific features.

### Links

- Use reference-style links for repeated URLs
- Descriptive link text (not "click here")
- Verify internal links point to existing files
- Use relative paths for internal documentation links

**Good:**
```markdown
See the [Configuration Guide](configuration.md) for details.

For more information, check the [Grafana documentation][grafana-docs].

[grafana-docs]: https://grafana.com/docs/
```

**Bad:**
```markdown
Click [here](configuration.md).

See https://grafana.com/docs/ for more info.
```

### Lists

- Consistent indentation (2 spaces for nested items)
- Blank line before/after lists
- Ordered lists for sequential steps
- Unordered lists for non-sequential items

**Good:**
```markdown
This is a paragraph.

- First item
- Second item
  - Nested item
  - Another nested item
- Third item

Another paragraph.
```

**Bad:**
```markdown
No spacing before list:
- Item 1
    - Inconsistent indentation (4 spaces)
- Item 2
No spacing after list.
```

### Admonitions and Notes

- Use blockquotes with bold labels for callouts
- Standard labels: **Note:**, **Important:**, **Warning:**, **Tip:**
- Format: `> **Label:** Message text`

**Example:**
```markdown
> **Note:** This is an informational note.

> **Important:** This requires attention.

> **Warning:** Be careful with this operation.

> **Tip:** Here's a helpful suggestion.
```

### Spacing

- One blank line between paragraphs
- One blank line before/after headers
- One blank line before/after code blocks
- One blank line before/after lists
- No trailing whitespace

### Common Language Tags

- `sh` - Shell commands (Alpine-compatible, default)
- `bash` - Bash-specific features (arrays, process substitution, etc.)
- `yaml` - YAML configuration
- `json` - JSON data
- `typescript` - TypeScript code
- `javascript` - JavaScript code
- `go` - Go code
- `sql` - SQL queries
- `ini` - INI config files
- `markdown` - Markdown examples
- `text` or `plaintext` - Plain text output

### File Paths

- Use inline code for file/directory paths: `` `/path/to/file` ``
- Use absolute paths when clarity requires
- Use relative paths in code examples

**Good:**
```markdown
Edit the `/docs/index.md` file.

Run the command from your project root: `./tests/`
```

### Command Examples

- Show full commands with all required flags
- Add comments for complex commands
- Show expected output when helpful
- Use descriptive prompts for shell sessions

**Good:**
````markdown
```bash
# Run K6 tests with verbose output
grafana-bench test \
  --test-runner k6 \
  --suite-name my-repo/k6-tests \
  --suite-path ./tests \
  --log-level debug
```

Expected output:
```text
Tests executed 5
Tests passed 5
Tests failed 0
```
````

### Version References

- Use semantic versioning: `vX.Y.Z`
- Reference version in code examples: `grafana-bench:`v1.0.1`
- Note: Version references are auto-updated by running `make docs`

## Content Style Guidelines

### Writing Style

- **Clear and concise**: Use simple, direct language
- **Active voice**: "Run the command" not "The command should be run"
- **Present tense**: "Bench provides" not "Bench will provide"
- **Second person**: Address the reader as "you"

### Code Examples

- **Complete examples**: Show all required setup, not just fragments
- **Working code**: Test all examples to ensure they work
- **Comments**: Explain non-obvious parts, but avoid over-commenting
- **Realistic**: Use realistic values (not `foo`, `bar`, `example.com`)

### Structure

- **Prerequisites section**: List requirements before main content
- **Step-by-step**: Number sequential steps clearly
- **Progressive complexity**: Start simple, add complexity gradually
- **Next steps**: Link to related docs at the end

## File Organization

### Documentation Files

- Use lowercase with hyphens: `getting-started.md`
- Auto-generated files use underscores: `bench_test.md` (DO NOT EDIT)
- Place all documentation in `/docs/` directory

### File Naming Conventions

- **Guides**: `writing-k6-tests.md`, `configuration-guide.md`
- **Reference**: `configuration-reference.md`, `cli-reference.md`
- **Tutorials**: `first-test.md`, `quickstart.md`
- **Lists**: `glossary.md`, `troubleshooting.md`, `cheat-sheet.md`

## Maintenance

### Auto-Generated Content

Files prefixed with `bench_` are auto-generated from CLI code:

- `bench.md`
- `bench_test.md`
- `bench_report.md`
- `bench_validate.md`
- `bench_checkout.md`
- `bench_version.md`

**DO NOT EDIT** these files directly. Changes will be overwritten when running `make docs`.

To modify auto-generated content, edit the corresponding source code in the `cmd/` directory.

### Version Updates

Version references throughout documentation are auto-updated:

```bash
make docs  # Updates version references and regenerates CLI docs
```

### Link Validation

Run the validation script to check for broken links and common issues:

```bash
./scripts/validate-docs.sh
```

## Checklist for New Documentation

Before submitting new documentation:

- [ ] Follows style guide formatting
- [ ] All code examples tested and working
- [ ] Internal links verified (no broken links)
- [ ] Appropriate language tags on code blocks
- [ ] No typos or grammar errors
- [ ] Single blank lines for spacing
- [ ] Admonitions use standard format
- [ ] Added to `mkdocs.yml` navigation
- [ ] Cross-links to related pages included
- [ ] Version references use correct format

## Examples

### Complete Document Template

```markdown
# Document Title

Brief introduction explaining what this document covers.

## Prerequisites

- Requirement 1
- Requirement 2

## Section 1

Content with examples.

```bash
# Example command
grafana-bench test \
  --suite-name my-repo/tests \
  --suite-path ./tests
```

> **Note:** Important information about this command.

## Section 2

More content.

### Subsection

Detailed information.

## Next Steps

- [Related Doc 1](link1.md)
- [Related Doc 2](link2.md)

## Related Pages

- [Guide 1](guide1.md) - Description
- [Guide 2](guide2.md) - Description
```

### Good vs Bad Examples

**Good Example:**
````markdown
## Running Your First Test

Follow these steps to run a basic K6 test:

1. Start Grafana:

```bash
docker run -d -p 3000:3000 grafana/grafana
```

2. Create a test file named `test.ts`:

```typescript
import { check } from 'k6';
import { http } from 'k6/http';

export default function () {
  const res = http.get('http://localhost:3000');
  check(res, { 'status is 200': (r) => r.status === 200 });
}
```

3. Run the test:

```bash
grafana-bench test \
  --suite-name my-repo/quickstart \
  --suite-path test.ts
```

> **Note:** The test connects to Grafana on port 3000 by default.
````

**Bad Example:**
````markdown
##Running Your First Test
Follow these steps to run a basic K6 test:
1. Start Grafana:
    docker run -d -p 3000:3000 grafana/grafana
2. Create a test file:
```
import { check } from 'k6';
```
3. Run the test.
````

## Questions?

If you have questions about documentation style:

- Check existing documentation for examples
- Ask in #grafana-bench Slack channel
- Refer to [MkDocs Material documentation](https://squidfunk.github.io/mkdocs-material/)
