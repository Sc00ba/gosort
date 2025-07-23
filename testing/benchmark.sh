#!/bin/bash

set -e

LARGE_FILE="large_file.txt"
GOSORT_OUTPUT="sorted_gosort.txt"
GNU_SORT_OUTPUT="sorted_gnu.txt"
WORD_SOURCE="/usr/share/dict/words"
LINE_COUNT=10000000
RECREATE_TEST_FILE=false
BUFFER_SIZE_MB=256
GO_MEM_LIMIT="300MiB"
USE_GO_MEM_LIMIT=false

for i in "$@"
do
case $i in
    --buffer-size=*)
    BUFFER_SIZE_MB="${i#*=}"
    shift
    ;;
    --go-mem-limit=*)
    GO_MEM_LIMIT="${i#*=}"
    USE_GO_MEM_LIMIT=true
    shift
    ;;
    --line-count=*)
    LINE_COUNT="${i#*=}"
    RECREATE_TEST_FILE=true
    shift
    ;;
    *)
    ;;
esac
done

BUFFER_SIZE_BYTES=$((BUFFER_SIZE_MB * 1024 * 1024))
BUFFER_SIZE_GNU="${BUFFER_SIZE_MB}M"

if command -v /usr/bin/time &> /dev/null; then
    USE_DETAILED_TIME=true
    TIME_FORMAT='Elapsed: %E | User: %U | Sys: %S | Max RSS: %M KB'
    echo "✅ Using /usr/bin/time for detailed memory measurement."
else
    USE_DETAILED_TIME=false
    echo "⚠️ /usr/bin/time not found. Memory usage will not be measured. Using basic 'time'."
fi
echo

echo "Building gosort executable..."
if ! go build -o gosort .; then
    echo "❌ Build failed. Please fix any compilation errors before running the benchmark."
    exit 1
fi
echo "✅ Build successful."
echo

if [ "$RECREATE_TEST_FILE" = true ] || [ ! -f "$LARGE_FILE" ]; then
    if [ "$RECREATE_TEST_FILE" = true ]; then
        echo "User specified --line-count. Forcing recreation of test file..."
    else
        echo "Large test file ('$LARGE_FILE') not found."
    fi

    if [ ! -f "$WORD_SOURCE" ]; then
        echo "❌ Word source file ('$WORD_SOURCE') not found. Cannot generate test file."
        exit 1
    fi

    echo "Generating a new test file with $LINE_COUNT lines..."
    shuf -r -n "$LINE_COUNT" "$WORD_SOURCE" > "$LARGE_FILE"
    echo "✅ Test file generated."
else
    echo "✅ Using existing test file: '$LARGE_FILE'."
fi
echo

(
  if [ "$USE_GO_MEM_LIMIT" = true ]; then
    echo "--- Benchmarking gosort with a ${GO_MEM_LIMIT} memory limit ---"
    if [ "$USE_DETAILED_TIME" = true ]; then
      GOMEMLIMIT="$GO_MEM_LIMIT" /usr/bin/time -f "$TIME_FORMAT" ./gosort -S "$BUFFER_SIZE_BYTES" < "$LARGE_FILE" > "$GOSORT_OUTPUT"
    else
      GOMEMLIMIT="$GO_MEM_LIMIT" time ./gosort -S "$BUFFER_SIZE_BYTES" < "$LARGE_FILE" > "$GOSORT_OUTPUT"
    fi
  else
    echo "--- Benchmarking gosort with no Go memory limit ---"
    if [ "$USE_DETAILED_TIME" = true ]; then
      /usr/bin/time -f "$TIME_FORMAT" ./gosort -S "$BUFFER_SIZE_BYTES" < "$LARGE_FILE" > "$GOSORT_OUTPUT"
    else
      time ./gosort -S "$BUFFER_SIZE_BYTES" < "$LARGE_FILE" > "$GOSORT_OUTPUT"
    fi
  fi
)
echo "✅ gosort finished. Output saved to '$GOSORT_OUTPUT'."
echo

echo "--- Benchmarking GNU sort with a ${BUFFER_SIZE_MB}MB buffer ---"
(
  if [ "$USE_DETAILED_TIME" = true ]; then
    LC_ALL=C /usr/bin/time -f "$TIME_FORMAT" sort -S "$BUFFER_SIZE_GNU" "$LARGE_FILE" > "$GNU_SORT_OUTPUT"
  else
    LC_ALL=C time sort -S "$BUFFER_SIZE_GNU" "$LARGE_FILE" > "$GNU_SORT_OUTPUT"
  fi
)
echo "✅ GNU sort finished. Output saved to '$GNU_SORT_OUTPUT'."
echo

echo "--- Verifying output ---"
if diff -q "$GOSORT_OUTPUT" "$GNU_SORT_OUTPUT"; then
    echo "✅ Success! The output from gosort and GNU sort is identical."
else
    echo "❌ Failure! The output files differ. There might be a bug in your sorting logic."
fi
echo

echo "Benchmark complete."
