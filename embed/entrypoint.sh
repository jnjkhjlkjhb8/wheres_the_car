#!/bin/sh
ollama serve &
until ollama list > /dev/null 2>&1; do sleep 2; done
ollama pull qwen3-embedding:0.6b
wait
