#!/usr/bin/env bash

set -euo pipefail

keys=$(redis-cli KEYS schedule:*)

for key in $keys; do
  value=$(redis-cli GET "$key")
  echo "$key ==> $value"
done
