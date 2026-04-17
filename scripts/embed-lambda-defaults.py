#!/usr/bin/env python3
"""Post-process a packaged CloudFormation template to embed Lambda S3 defaults.

Usage:
    python3 scripts/embed-lambda-defaults.py TEMPLATE_FILE PARAM=VALUE [PARAM=VALUE ...]

Example:
    python3 scripts/embed-lambda-defaults.py .aws-sam/build/regions/us-west-2/logwriter.yaml \
        LambdaS3BucketPrefix=observeinc \
        LambdaS3Key=aws-sam-apps/v1.19.2/subscriber.zip

This updates the Default value for each named parameter in the template's
Parameters section, writing the result back to the same file. Parameters not
found in the template are silently skipped.
"""

import re
import sys


def embed_defaults(template_path, defaults):
    with open(template_path, "r") as f:
        content = f.read()

    for param_name, param_value in defaults.items():
        # Match the parameter block and update its Default line.
        # Pattern: find "  ParamName:\n" followed by lines until "    Default: '...'"
        # and replace the default value.
        pattern = re.compile(
            r"(^  " + re.escape(param_name) + r":\s*\n"
            r"(?:    .*\n)*?"       # non-greedy match of indented lines
            r"    Default: )'[^']*'",
            re.MULTILINE,
        )
        replacement = r"\g<1>'" + param_value.replace("\\", "\\\\") + "'"
        new_content = pattern.sub(replacement, content)

        if new_content == content:
            # Try double-quoted default
            pattern_dq = re.compile(
                r"(^  " + re.escape(param_name) + r":\s*\n"
                r"(?:    .*\n)*?"
                r'    Default: )"[^"]*"',
                re.MULTILINE,
            )
            replacement_dq = r'\g<1>"' + param_value + '"'
            new_content = pattern_dq.sub(replacement_dq, content)

        if new_content == content:
            # No Default line exists — insert one after the last indented
            # property line in the parameter block.
            insert_pattern = re.compile(
                r"(^  " + re.escape(param_name) + r":\s*\n"
                r"(?:    .*\n)*?)"     # capture the full param block
                r"(?=  \S|\Z)",        # stop at the next top-level key or EOF
                re.MULTILINE,
            )
            match = insert_pattern.search(content)
            if match:
                block = match.group(1)
                new_block = block + "    Default: '" + param_value + "'\n"
                new_content = content[:match.start()] + new_block + content[match.end():]

        content = new_content

    with open(template_path, "w") as f:
        f.write(content)


def main():
    if len(sys.argv) < 3:
        print(__doc__, file=sys.stderr)
        sys.exit(1)

    template_path = sys.argv[1]
    defaults = {}
    for arg in sys.argv[2:]:
        if "=" not in arg:
            print(f"Invalid argument (expected PARAM=VALUE): {arg}", file=sys.stderr)
            sys.exit(1)
        key, value = arg.split("=", 1)
        defaults[key] = value

    embed_defaults(template_path, defaults)
    print(f"Embedded {len(defaults)} default(s) in {template_path}")


if __name__ == "__main__":
    main()
