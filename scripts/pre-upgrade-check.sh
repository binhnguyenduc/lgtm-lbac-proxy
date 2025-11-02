#!/bin/bash

# Pre-upgrade check script for LGTM LBAC Proxy v0.12.0+
# This script validates that labels.yaml is in the required extended format

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

LABELS_FILE="${LABELS_FILE:-$PROJECT_ROOT/configs/labels.yaml}"
MIGRATE_TOOL="$PROJECT_ROOT/cmd/migrate-labels/migrate-labels"

echo "=== LGTM LBAC Proxy Pre-Upgrade Check ==="
echo

# Check if labels.yaml exists
if [ ! -f "$LABELS_FILE" ]; then
    echo "❌ ERROR: labels.yaml not found at: $LABELS_FILE"
    echo "   Set LABELS_FILE environment variable to specify a different path"
    exit 1
fi

echo "📄 Checking labels file: $LABELS_FILE"
echo

# Check if migration tool exists, build if needed
if [ ! -f "$MIGRATE_TOOL" ]; then
    echo "🔨 Building migration tool..."
    (cd "$PROJECT_ROOT/cmd/migrate-labels" && go build -o migrate-labels .)
    if [ $? -ne 0 ]; then
        echo "❌ Failed to build migration tool"
        exit 1
    fi
fi

# Run validation
if "$MIGRATE_TOOL" -input "$LABELS_FILE" -validate; then
    echo
    echo "========================================="
    echo "✅ SUCCESS: Ready to upgrade to v0.12.0+"
    echo "========================================="
    exit 0
else
    echo
    echo "========================================="
    echo "❌ FAILED: Migration required"
    echo "========================================="
    echo
    echo "To migrate your labels.yaml:"
    echo "  1. Backup current file:"
    echo "     cp $LABELS_FILE ${LABELS_FILE}.backup"
    echo
    echo "  2. Run migration:"
    echo "     $MIGRATE_TOOL -input $LABELS_FILE -output ${LABELS_FILE}.new"
    echo
    echo "  3. Review and replace:"
    echo "     mv ${LABELS_FILE}.new $LABELS_FILE"
    echo
    echo "See cmd/migrate-labels/README.md for detailed instructions"
    exit 1
fi
