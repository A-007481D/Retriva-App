#!/usr/bin/env bash
# inject-cookies.sh — Injects newly exported Instagram cookies into Retriva
set -e

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TARGET_COOKIES="$REPO_ROOT/data/cookies.txt"

mkdir -p "$REPO_ROOT/data"

SRC_COOKIE="$1"

# If no argument is given, find the most recently downloaded cookie file in ~/Downloads
if [ -z "$SRC_COOKIE" ]; then
    SRC_COOKIE="$(ls -t "$HOME/Downloads"/*cookie*.txt 2>/dev/null | head -n 1 || true)"
    if [ -z "$SRC_COOKIE" ]; then
        SRC_COOKIE="$(ls -t "$HOME/Downloads"/*instagram*.txt 2>/dev/null | head -n 1 || true)"
    fi
fi

if [ -z "$SRC_COOKIE" ] || [ ! -f "$SRC_COOKIE" ]; then
    echo "❌ Error: No cookie file found!"
    echo "Usage: ./inject-cookies.sh [path/to/cookies.txt]"
    echo "Or download a cookies.txt file to ~/Downloads/ and run ./inject-cookies.sh"
    exit 1
fi

if [ ! -s "$SRC_COOKIE" ]; then
    echo "❌ Error: '$SRC_COOKIE' is empty (0 bytes)."
    exit 1
fi

SRC_REAL="$(realpath "$SRC_COOKIE" 2>/dev/null || echo "$SRC_COOKIE")"
TARGET_REAL="$(realpath "$TARGET_COOKIES" 2>/dev/null || echo "$TARGET_COOKIES")"

if [ "$SRC_REAL" != "$TARGET_REAL" ]; then
    cp -f "$SRC_COOKIE" "$TARGET_COOKIES"
fi

FILE_SIZE=$(wc -c < "$TARGET_COOKIES" | tr -d ' ')
SESSION_COUNT=$(grep -c "sessionid" "$TARGET_COOKIES" || true)

echo "✅ Successfully injected cookies into Retriva!"
echo "   Source: $SRC_COOKIE"
echo "   Target: $TARGET_COOKIES"
echo "   Size:   $FILE_SIZE bytes"

if [ "$SESSION_COUNT" -gt 0 ]; then
    echo "   Status: sessionid cookie found ✓"
else
    echo "   ⚠️ Warning: 'sessionid' cookie not found in file. Make sure you were logged in to Instagram before exporting."
fi

echo ""
echo "🚀 Retriva picks this up immediately on your next download or inspection (no restart required)."
