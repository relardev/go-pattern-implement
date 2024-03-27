#!/bin/bash
#
if [ $# -ne 3 ]; then
    echo "Usage: $0 <implementation_name> <test_name> <input_file>"
    exit 1
fi

implementation_name=$1
test_name=$2
input_file=$3

mkdir -p test/$test_name
cp $input_file test/$test_name/input
./test/update_expected.sh $test_name $implementation_name
