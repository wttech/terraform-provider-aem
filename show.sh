#!/bin/bash

sh run.sh examples/ssh show -json | python -m json.tool
