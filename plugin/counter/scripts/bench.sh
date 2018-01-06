#!/bin/bash
cd .. && time go test -v -benchmem -bench . | tee ./scripts/benchmarks.txt