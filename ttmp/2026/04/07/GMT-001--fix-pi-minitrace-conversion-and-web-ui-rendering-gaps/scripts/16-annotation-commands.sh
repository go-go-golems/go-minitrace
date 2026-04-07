#!/bin/bash
# Commands used to annotate the session with go-minitrace
# These were run from the output directory where ./analysis/ contains the minitrace archive

SESSION=f6498c9d-3c41-4850-8f9c-667eca2ee271

# List annotations
go-minitrace annotate list --output-dir ./analysis --session $SESSION

# Add annotation (examples from the session)
go-minitrace annotate add \
  --output-dir ./analysis \
  --session $SESSION \
  --category observation \
  --title "PPPP-001: Ferrari SDK build fix - EvdevPenReader" \
  --annotator analysis

go-minitrace annotate add \
  --output-dir ./analysis \
  --session $SESSION \
  --category success \
  --title "Committed EvdevPenReader - pen input now works on device" \
  --tags git-commit,evdev,pen-input,success \
  --annotator analysis

# Sync annotations back to JSON
go-minitrace annotate sync --output-dir ./analysis --session $SESSION

# Validate the archive
go-minitrace validate --path ./analysis --recursive
