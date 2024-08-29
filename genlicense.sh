#!/bin/sh
rm -r licenses
set -e
for p in $(cd vendor && find -iname '*license*' -or -iname '*notice*' -or -iname '*patent*'); do
	install -D vendor/$p licenses/$p
done
