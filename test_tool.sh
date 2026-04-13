#!/bin/bash
# Usage: ./test_tool.sh <tool_name> '<json_args>'
# Example: ./test_tool.sh get_dictionaries '{"dictionary_names":"Currencies"}'

TOOL=$1
ARGS=$2

SSE_OUT=$(mktemp)
curl -sN http://localhost:8080/sse > "$SSE_OUT" 2>&1 &
SSE_PID=$!
sleep 1

SESSION_URL=$(grep "^data:" "$SSE_OUT" | head -1 | sed 's/^data: //')

curl -s -X POST "http://localhost:8080${SESSION_URL}" \
  -H "Content-Type: application/json" \
  -d "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"tools/call\",\"params\":{\"name\":\"$TOOL\",\"arguments\":$ARGS}}"

sleep 3

# Extract result from SSE stream (skip the endpoint event line)
grep "^data:" "$SSE_OUT" | tail -1 | sed 's/^data: //'

kill $SSE_PID 2>/dev/null
rm "$SSE_OUT"
