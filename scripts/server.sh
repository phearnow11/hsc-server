#!/bin/bash

if [ ! -f .air.toml ]; then
  echo "File .air.toml not found! Init"
  air init
fi
