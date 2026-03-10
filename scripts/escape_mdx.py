#!/usr/bin/env python3
"""Post-processes MDX files to escape bare { } < > outside code spans/fenced blocks.

MDX parses {…} as JavaScript expressions and <…> as JSX/HTML tags.
Proto field comments may contain syntax like ConfigsFilter{ids: [...]} or
comparisons like <= that cause MDX parse errors.

Usage:
    python3 escape_mdx.py <file>  # in-place
    python3 escape_mdx.py         # stdin → stdout
"""
import re
import sys

_CODE_SPAN = re.compile(r'(`+.+?`+)')
_FENCE = re.compile(r'^[ ]{0,3}(`{3,}|~{3,})')
# Match markdown links [text](url) and image links ![alt](url) to preserve them
_MD_LINK = re.compile(r'!?\[[^\]]*\]\([^)]*\)')
# Match HTML entities like &lt; &gt; &amp; to avoid double-escaping
_HTML_ENTITY = re.compile(r'&[a-zA-Z]+;|&#\d+;|&#x[0-9a-fA-F]+;')


def _escape_segment(text: str) -> str:
    """Escape { } < > in a plain text segment (not inside code spans or markdown links)."""
    # Split out markdown links and HTML entities to preserve them
    parts = []
    last = 0
    for m in re.finditer(r'(!?\[[^\]]*\]\([^)]*\))|(&[a-zA-Z]+;|&#\d+;|&#x[0-9a-fA-F]+;)', text):
        if m.start() > last:
            parts.append(('text', text[last:m.start()]))
        parts.append(('preserve', m.group()))
        last = m.end()
    if last < len(text):
        parts.append(('text', text[last:]))

    result = []
    for kind, segment in parts:
        if kind == 'preserve':
            result.append(segment)
        else:
            segment = segment.replace('{', r'\{').replace('}', r'\}')
            segment = segment.replace('<=', '≤').replace('>=', '≥')
            segment = segment.replace('<', '&lt;').replace('>', '&gt;')
            result.append(segment)
    return ''.join(result)


def escape_line(line: str) -> str:
    """Escape { } < > outside inline code spans."""
    parts = _CODE_SPAN.split(line)
    return ''.join(
        part if i % 2 else _escape_segment(part)
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
