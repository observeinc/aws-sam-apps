#!/bin/bash
# Ensure every integration test file declares the tagging provider block.
# Without it, `default_tags` do not propagate to `run` blocks that use
# external modules, and the IAM policy at integration/github-actions-role-policy.json
# will deny operations on resources it expects to be tagged.
set -euo pipefail

REQUIRED='managed-by" = "integration-test"'
MISSING=()

for f in integration/tests/*.tftest.hcl; do
  if ! grep -qF "$REQUIRED" "$f"; then
    MISSING+=("$f")
  fi
done

if [ ${#MISSING[@]} -gt 0 ]; then
  echo "ERROR: the following test files are missing the required provider block" >&2
  echo "with default_tags '\"managed-by\" = \"integration-test\"':" >&2
  printf '  %s\n' "${MISSING[@]}" >&2
  echo >&2
  echo "Add this at the top of each file:" >&2
  cat >&2 <<'EOF'

provider "aws" {
  default_tags {
    tags = {
      "managed-by" = "integration-test"
    }
  }
}

EOF
  exit 1
fi

echo "OK: all $(ls integration/tests/*.tftest.hcl | wc -l | tr -d ' ') integration test files declare the tagging provider block."
