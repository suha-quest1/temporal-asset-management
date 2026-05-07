#!/bin/bash
find /tmp/assetmgmt -type f \( -name "*.go" -o -name "*.py" \) | while read f; do
  rel=${f#/tmp/assetmgmt/}
  if [ -f "/private/tmp/assetmgmt/$rel" ]; then
    git diff --no-index "$f" "/private/tmp/assetmgmt/$rel"
  else
    echo "Missing in /private/tmp/assetmgmt: $rel"
  fi
done
