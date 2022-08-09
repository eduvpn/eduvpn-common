#!/usr/bin/env bash

# Check if wg show has any output
output_wg="$(wg show)"
[[ -n $output_wg ]]
