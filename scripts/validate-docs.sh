#!/bin/bash
# Validate documentation quality

set -e

echo "🔍 Checking Grafana Bench documentation..."
echo ""

# Track overall status
ERRORS=0
WARNINGS=0

# Change to project root
cd "$(dirname "$0")/.."

# Check for broken internal links
echo "📎 Checking internal links..."
BROKEN_LINKS=0
while IFS= read -r file; do
    # Extract markdown links [text](file.md)
    grep -oE '\[([^\]]+)\]\(([^)]+\.md[^)]*)\)' "$file" 2>/dev/null | while read -r link; do
        # Extract the file path from the link
        link_path=$(echo "$link" | sed -E 's/.*\(([^)]+)\).*/\1/' | sed 's/#.*//')

        # Skip external links
        if [[ "$link_path" =~ ^https?:// ]]; then
            continue
        fi

        # Construct absolute path
        dir=$(dirname "$file")
        if [[ "$link_path" == /* ]]; then
            # Absolute path from docs root
            target="docs/$link_path"
        else
            # Relative path
            target="$dir/$link_path"
        fi

        # Check if target exists
        if [ ! -f "$target" ]; then
            echo "  ❌ Broken link in $file: $link_path"
            ((BROKEN_LINKS++))
        fi
    done
done < <(find docs -name "*.md" -type f)

if [ $BROKEN_LINKS -eq 0 ]; then
    echo "  ✅ No broken internal links"
else
    echo "  ❌ Found $BROKEN_LINKS broken link(s)"
    ((ERRORS+=BROKEN_LINKS))
fi
echo ""

# Check for common typos
echo "📝 Checking for common typos..."
TYPOS=$(grep -rn "becnch\|princples\|cusstom\|Broswer" docs/*.md 2>/dev/null | grep -v "bench_" || true)
if [ -z "$TYPOS" ]; then
    echo "  ✅ No common typos found"
else
    echo "  ❌ Found typos:"
    echo "$TYPOS" | sed 's/^/    /'
    TYPO_COUNT=$(echo "$TYPOS" | wc -l)
    ((ERRORS+=TYPO_COUNT))
fi
echo ""

# Check for indented code blocks (should use fenced)
# Note: This check may have false positives for YAML indentation inside fenced blocks
echo "💻 Checking for indented code blocks..."
INDENTED=$(grep -rn "^    [a-z]" docs/*.md 2>/dev/null | grep -v "bench_" | grep -v "\.yaml" | head -5 || true)
if [ -z "$INDENTED" ]; then
    echo "  ✅ No indented code blocks found (note: may have false positives for YAML indentation)"
else
    echo "  ⚠️  Possible indented code blocks (may be false positives for YAML):"
    echo "$INDENTED" | sed 's/^/    /'
    echo "  (Note: Lines inside fenced YAML blocks are OK)"
fi
echo ""

# Check for inconsistent language tags
echo "🏷️  Checking for inconsistent language tags..."
INCONSISTENT=$(grep -rn "\`\`\`shell\|\`\`\`yml" docs/*.md 2>/dev/null || true)
if [ -z "$INCONSISTENT" ]; then
    echo "  ✅ Consistent language tags"
else
    echo "  ⚠️  Found inconsistent language tags (use 'sh' or 'bash', not 'shell'; use 'yaml' not 'yml'):"
    echo "$INCONSISTENT" | sed 's/^/    /'
    INCONSISTENT_COUNT=$(echo "$INCONSISTENT" | wc -l)
    ((WARNINGS+=INCONSISTENT_COUNT))
fi
echo ""

# Check for "Documentation coming soon" placeholders
echo "📋 Checking for incomplete sections..."
INCOMPLETE=$(grep -rn "Documentation coming soon\|TODO\|FIXME" docs/*.md 2>/dev/null | grep -v "bench_" || true)
if [ -z "$INCOMPLETE" ]; then
    echo "  ✅ No incomplete sections found"
else
    echo "  ⚠️  Found incomplete sections:"
    echo "$INCOMPLETE" | sed 's/^/    /'
    INCOMPLETE_COUNT=$(echo "$INCOMPLETE" | wc -l)
    ((WARNINGS+=INCOMPLETE_COUNT))
fi
echo ""

# Check for trailing whitespace
echo "🧹 Checking for trailing whitespace..."
TRAILING=$(grep -rn " $" docs/*.md 2>/dev/null | grep -v "bench_" | head -5 || true)
if [ -z "$TRAILING" ]; then
    echo "  ✅ No trailing whitespace"
else
    echo "  ⚠️  Found trailing whitespace (showing first 5):"
    echo "$TRAILING" | sed 's/^/    /'
    TRAILING_COUNT=$(grep -rc " $" docs/*.md 2>/dev/null | grep -v ":0$" | grep -v "bench_" | wc -l)
    ((WARNINGS+=TRAILING_COUNT))
fi
echo ""

# Summary
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📊 Validation Summary"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
if [ $ERRORS -eq 0 ] && [ $WARNINGS -eq 0 ]; then
    echo "✅ All checks passed! Documentation quality is excellent."
    exit 0
elif [ $ERRORS -eq 0 ]; then
    echo "⚠️  No errors, but found $WARNINGS warning(s)"
    echo "   Consider fixing warnings for improved documentation quality."
    exit 0
else
    echo "❌ Found $ERRORS error(s) and $WARNINGS warning(s)"
    echo "   Please fix errors before proceeding."
    exit 1
fi
