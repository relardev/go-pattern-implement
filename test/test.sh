#!/bin/bash

cat test/prometheus/input | ./bin/go-component-generator implement --package abc prometheus > test/prometheus/result
# cat test/prometheus/input | ./bin/go-component-generator implement --package abc prometheus > test/prometheus/expected

result="test/prometheus/result"
expected="test/prometheus/expected"

diff_output=$(diff "$result" "$expected")

if [ $? -ne 0 ]; then
    echo "result is different from expected: $result vs $expected"
    echo "$diff_output"
    exit 1
fi
