#!/bin/bash

if [ $# -ne 2 ]; then
    echo "Usage: $0 <test_name> <implementation_name>"
    exit 1
fi

test_name=$1
implementation_name=$2

cat test/$test_name/input | ./bin/go-component-generator implement --package abc $implementation_name > test/$test_name/expected
