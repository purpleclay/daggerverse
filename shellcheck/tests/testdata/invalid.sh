#!/bin/sh

# Arrays are not defined in POSIX shell
items=("apple" "banana" "cherry" "date" "elderberry")
for item in "${items[@]}"
do
    echo "item: $item"
done
