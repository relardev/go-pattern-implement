#!/bin/bash

tests="prometheus cache semaphore throttle-error throttle filter filter-error"

for test_name in $tests; do
    echo "Testing: $test_name"
    cat test/$test_name/input | ./bin/go-component-generator implement --package abc $test_name > test/$test_name/result

    result="test/$test_name/result"
    expected="test/$test_name/expected"

    diff_output=$(diff "$result" "$expected")

    if [ $? -ne 0 ]; then
        echo "result is different from expected: $result vs $expected"
        echo "$diff_output"
        exit 1
    fi
done
