#!/bin/sh
# Use installed BasicTeX fonts and plain code blocks on this Mac.
exec /opt/homebrew/bin/pandoc --no-highlight --toc --toc-depth=2 -V fontsize=11pt "$@"
