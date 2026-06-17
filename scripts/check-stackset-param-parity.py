#!/usr/bin/env python3
"""Check that each StackSet wrapper template's pass-through parameters
mirror the parameter constraints declared in the underlying app template.

For every (inner template, wrapper template) pair, this script compares
the constraint properties (AllowedPattern, AllowedValues, MinValue,
MaxValue, MaxLength, MinLength) of every parameter that appears in both.

Outcomes:
  WARN: a parameter exists in the inner template but not in the wrapper
        (the wrapper deliberately doesn't surface this param). Does not
        fail the check.
  FAIL: a parameter exists in both, but the wrapper omits a constraint
        the inner declares, or the constraints have different values.
        These let invalid input through wrapper-side validation, only to
        be caught by the inner stack instance -- which under StackSets'
        FailureTolerancePercentage=100 default surfaces as a silent
        per-instance failure instead of a fast fail at deploy time.

Exits non-zero only on FAIL conditions.
"""

import re
import sys

CONSTRAINTS = [
    "AllowedPattern",
    "AllowedValues",
    "MinValue",
    "MaxValue",
    "MaxLength",
    "MinLength",
]

PAIRS = [
    ("apps/logwriter/template.yaml",    "apps/logwriter-stackset/template.yaml"),
    ("apps/metricstream/template.yaml", "apps/metricstream-stackset/template.yaml"),
    ("apps/externalrole/template.yaml", "apps/externalrole-stackset/template.yaml"),
]


def parse_param_blocks(path):
    """Return {param_name: param_block_text} for every parameter under
    the top-level Parameters: section. Each block is the lines indented
    under the parameter name, joined back with newlines."""
    with open(path) as f:
        lines = f.read().split("\n")

    blocks = {}
    in_params = False
    cur_name = None
    cur_lines = []
    for line in lines:
        if line.rstrip() == "Parameters:":
            in_params = True
            continue
        if in_params and re.match(r"^[A-Z]", line):
            # Top-level section after Parameters: ends the block.
            in_params = False
            if cur_name:
                blocks[cur_name] = "\n".join(cur_lines)
                cur_name = None
            break
        if not in_params:
            continue
        m = re.match(r"^  ([A-Z][a-zA-Z]*):\s*$", line)
        if m:
            if cur_name:
                blocks[cur_name] = "\n".join(cur_lines)
            cur_name = m.group(1)
            cur_lines = []
        else:
            cur_lines.append(line)
    if cur_name:
        blocks[cur_name] = "\n".join(cur_lines)
    return blocks


def constraint_value(block, key):
    """Extract a single constraint's value from a parameter block.
    Returns None if absent. Inline values are returned as a stripped
    string. List values (e.g. AllowedValues across several lines) are
    normalized to a sorted bracketed string for comparison."""
    # Try multi-line list form first (must come before the inline check
    # because both can match an empty inline value):
    #     AllowedValues:
    #       - 'foo'
    #       - 'bar'
    list_match = re.search(
        r"^    " + key + r":\s*$(?:\n      - [^\n]*)+",
        block,
        re.MULTILINE,
    )
    if list_match:
        items = re.findall(r"^      - (.*)$", list_match.group(0), re.MULTILINE)
        return "[" + ", ".join(sorted(item.strip() for item in items)) + "]"
    # Inline form: `    AllowedPattern: '^foo$'`
    inline = re.search(r"^    " + key + r":\s*([^\s].*)$", block, re.MULTILINE)
    if inline:
        return inline.group(1).strip()
    return None


def main():
    warn_count = 0
    fail_count = 0
    for inner_path, wrapper_path in PAIRS:
        inner = parse_param_blocks(inner_path)
        wrapper = parse_param_blocks(wrapper_path)
        app = inner_path.split("/")[1]
        for name in inner:
            if name not in wrapper:
                print(
                    f"WARN  {app}.{name}: in inner but not in wrapper "
                    f"(wrapper does not surface this parameter)",
                    file=sys.stderr,
                )
                warn_count += 1
                continue
            for key in CONSTRAINTS:
                inner_val = constraint_value(inner[name], key)
                wrap_val = constraint_value(wrapper[name], key)
                if inner_val == wrap_val:
                    continue
                if inner_val is not None and wrap_val is None:
                    print(
                        f"FAIL  {app}.{name}: wrapper missing {key} "
                        f"(inner has {key}={inner_val})",
                        file=sys.stderr,
                    )
                    fail_count += 1
                elif inner_val is None and wrap_val is not None:
                    print(
                        f"WARN  {app}.{name}: wrapper has {key}={wrap_val} "
                        f"but inner has none (wrapper is stricter)",
                        file=sys.stderr,
                    )
                    warn_count += 1
                else:
                    print(
                        f"FAIL  {app}.{name}: {key} mismatch: "
                        f"inner={inner_val} wrapper={wrap_val}",
                        file=sys.stderr,
                    )
                    fail_count += 1

    if warn_count or fail_count:
        print(
            f"\n{warn_count} warning(s), {fail_count} failure(s)",
            file=sys.stderr,
        )
    if fail_count:
        sys.exit(1)
    print("OK: stackset parameter constraint parity check passed")


if __name__ == "__main__":
    main()
