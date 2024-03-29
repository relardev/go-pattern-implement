#!/bin/bash

tests='
prometheus
cache
semaphore
throttle-error
throttle
filter 
filter-error 
filter-return:filter-return-list
filter-return:filter-return-map
filter-param
'

for test in $tests; do
    echo $test | grep ":" > /dev/null
    if [ $? -eq 0 ]; then
        implementation=$(echo $test | cut -d ":" -f 1)
        test_dir=$(echo $test | cut -d ":" -f 2)
    else
        implementation=$test
        test_dir=$test
    fi
    echo "Testing implementation: $implementation, with test: $test_dir" 

    rm -f test/$test_dir/result

    cat test/$test_dir/input | ./bin/go-pattern-implement implement --package abc $implementation > test/$test_dir/result

    result="test/$test_dir/result"
    expected="test/$test_dir/expected"

    diff_output=$(diff "$result" "$expected")

    if [ $? -ne 0 ]; then
        echo "result is different from expected: $result vs $expected"
        echo "$diff_output"
        exit 1
    fi
done
