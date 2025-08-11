#!/usr/bin/env bash
set -euo pipefail

LARGE_FILE="input_file.bmf"
WORD_SOURCE="/usr/share/dict/words"
LINE_COUNT=10000000

for i in "$@"; do
  case $i in
    --line-count=*)
      LINE_COUNT="${i#*=}"
      shift
      ;;
    --word-source=*)
      WORD_SOURCE="${i#*=}"
      shift
      ;;
    --output=*)
      LARGE_FILE="${i#*=}"
      shift
      ;;
    *)
      ;;
  esac
done

if ! command -v shuf >/dev/null 2>&1; then
  echo "❌ 'shuf' not found. Please install coreutils (e.g., 'sudo apt-get install coreutils')."
  exit 1
fi

if [ ! -f "$WORD_SOURCE" ]; then
  echo "❌ Word source file ('$WORD_SOURCE') not found. Cannot generate test file."
  exit 1
fi

echo "Generating a new test file with $LINE_COUNT lines from '$WORD_SOURCE' → '$LARGE_FILE'..."
shuf -r -n "$LINE_COUNT" "$WORD_SOURCE" > "$LARGE_FILE"
echo "✅ Test file generated at '$LARGE_FILE'."

