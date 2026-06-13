#!/bin/sh
ollama serve &
OLLAMA_PID=$!
until ollama list > /dev/null 2>&1; do sleep 2; done
ollama pull qwen3-embedding:0.6b
trap "kill $OLLAMA_PID" TERM INT
wait $OLLAMA_PID
