#!/bin/bash

if [ $# -ne 1 ]; then
    echo "Usage: $0 <test_name>"
    exit 1
fi

test_name=$1

cat test/$test_name/input | ./bin/go-component-generator implement --package abc $test_name > test/$test_name/expected
