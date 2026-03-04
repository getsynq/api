#!/usr/bin/env python3
"""Post-processes MDX files to escape bare { and } outside code spans/fenced blocks.

MDX parses {…} as JavaScript expressions. Proto field comments may contain
code-like syntax (e.g. ConfigsFilter{ids: [...]}) that causes MDX/acorn parse errors.

Usage:
    python3 escape_mdx.py <file>  # in-place
    python3 escape_mdx.py         # stdin → stdout
"""
import re
import sys

_CODE_SPAN = re.compile(r'(`+.+?`+)')
_FENCE = re.compile(r'^[ ]{0,3}(`{3,}|~{3,})')


def escape_line(line: str) -> str:
    """Escape { and } outside inline code spans."""
    parts = _CODE_SPAN.split(line)
    return ''.join(
        part if i % 2 else part.replace('{', r'\{').replace('}', r'\}')
        for i, part in enumerate(parts)
    )


def process(text: str) -> str:
    output = []
    fence_marker = None  # None = outside fenced block
    for line in text.splitlines():
        m = _FENCE.match(line)
        if fence_marker is None and m:
            fence_marker = m.group(1)
            output.append(line)
        elif fence_marker and m and m.group(1)[0] == fence_marker[0] and len(m.group(1)) >= len(fence_marker):
            fence_marker = None
            output.append(line)
        elif fence_marker is None:
            output.append(escape_line(line))
        else:
            output.append(line)
    return '\n'.join(output) + ('\n' if text.endswith('\n') else '')


if __name__ == '__main__':
    if len(sys.argv) > 1:
        path = sys.argv[1]
        with open(path, 'r', encoding='utf-8') as f:
            content = f.read()
        with open(path, 'w', encoding='utf-8') as f:
            f.write(process(content))
    else:
        sys.stdout.write(process(sys.stdin.read()))
