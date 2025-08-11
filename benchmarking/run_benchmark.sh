#!/usr/bin/env bash
set -euo pipefail

LARGE_FILE="input_file.bmf"
GOSORT_OUTPUT="output_gosort.bmf"
GNU_SORT_OUTPUT="output_gnu_sort.bmf"
GO_MEM_LIMIT="300MiB"
USE_GO_MEM_LIMIT=false
GO_SORT_CHUNK_SIZE=$((128*1024*1024))

for i in "$@"; do
  case $i in
    --file=*)
      LARGE_FILE="${i#*=}"
      shift
      ;;
    --gosort-output=*)
      GOSORT_OUTPUT="${i#*=}"
      shift
      ;;
    --gnu-output=*)
      GNU_SORT_OUTPUT="${i#*=}"
      shift
      ;;
    --go-sort-chunk-size=*)
      GO_SORT_CHUNK_SIZE="${i#*=}"
      shift
      ;;
    --go-mem-limit=*)
      GO_MEM_LIMIT="${i#*=}"
      USE_GO_MEM_LIMIT=true
      shift
      ;;
    *)
      ;;
  esac
done

if command -v /usr/bin/time &>/dev/null; then
  USE_DETAILED_TIME=true
  TIME_FORMAT='Elapsed: %E | User: %U | Sys: %S | Max RSS: %M KB'
  echo "✅ Using /usr/bin/time for detailed memory measurement."
else
  USE_DETAILED_TIME=false
  echo "⚠️ /usr/bin/time not found. Memory usage will not be measured. Using basic 'time'."
fi
echo

if [ ! -f "$LARGE_FILE" ]; then
  echo "❌ Large test file ('$LARGE_FILE') not found."
  echo "   Generate it first, e.g.: ./generate_test_file.sh --output=$LARGE_FILE"
  exit 1
fi

# Build gosort
echo "Building gosort executable..."
if ! go build -o gosort .; then
  echo "❌ Build failed. Please fix compilation errors before running the benchmark."
  exit 1
fi
echo "✅ Build successful."
echo

# Run gosort benchmark
(
  if [ "$USE_GO_MEM_LIMIT" = true ]; then
    echo "--- Benchmarking gosort with a ${GO_MEM_LIMIT} memory limit ---"
    if [ "$USE_DETAILED_TIME" = true ]; then
      GOMEMLIMIT="$GO_MEM_LIMIT" /usr/bin/time -f "$TIME_FORMAT" ./gosort -cs "$GO_SORT_CHUNK_SIZE" < "$LARGE_FILE" > "$GOSORT_OUTPUT"
    else
      GOMEMLIMIT="$GO_MEM_LIMIT" time ./gosort -cs "$GO_SORT_CHUNK_SIZE" < "$LARGE_FILE" > "$GOSORT_OUTPUT"
    fi
  else
    echo "--- Benchmarking gosort with no Go memory limit ---"
    if [ "$USE_DETAILED_TIME" = true ]; then
      /usr/bin/time -f "$TIME_FORMAT" ./gosort -cs "$GO_SORT_CHUNK_SIZE" < "$LARGE_FILE" > "$GOSORT_OUTPUT"
    else
      time ./gosort -cs "$GO_SORT_CHUNK_SIZE" < "$LARGE_FILE" > "$GOSORT_OUTPUT"
    fi
  fi
)
echo "✅ gosort finished. Output saved to '$GOSORT_OUTPUT'."
echo

# Run GNU sort benchmark
echo "--- Benchmarking GNU sort ---"
(
  if [ "$USE_DETAILED_TIME" = true ]; then
    LC_ALL=C /usr/bin/time -f "$TIME_FORMAT" sort "$LARGE_FILE" > "$GNU_SORT_OUTPUT"
  else
    LC_ALL=C time sort "$LARGE_FILE" > "$GNU_SORT_OUTPUT"
  fi
)
echo "✅ GNU sort finished. Output saved to '$GNU_SORT_OUTPUT'."
echo

# Verify
echo "--- Verifying output ---"
if diff -q "$GOSORT_OUTPUT" "$GNU_SORT_OUTPUT"; then
  echo "✅ Success! The output from gosort and GNU sort is identical."
else
  echo "❌ Failure! The output files differ. There might be a bug in your sorting logic."
fi
echo

echo "Benchmark complete."

