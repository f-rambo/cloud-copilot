#!/bin/bash

for file in $(find internal/biz -name "*pb.go"); do
    cp -r "$file" "${file}.bak"

    if grep -qE '^type [A-Za-z_][A-Za-z0-9_]* struct \{' "$file.bak"; then
        awk '
    BEGIN {
        in_import = 0
    }
    /^import \(/ {
        in_import = 1
        print      # import (
        next
    }
    in_import && /^\)/ {
        in_import = 0
        if (!found_gorm) {
            print "    \"gorm.io/gorm\""
        }
        print      # )
        next
    }
    in_import {
        print
        if ($0 ~ /gorm\.io\/gorm/) {
            found_gorm = 1
        }
        next
    }
    /type [A-Za-z0-9_]+ struct {/ {
        print
        print "    gorm.Model"
        next
    }
    { print }
    ' "$file.bak" >"$file"
        go fmt "$file"
    fi
    rm -f "$file.bak"
done
