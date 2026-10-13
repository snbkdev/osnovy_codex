#!/usr/bin/env sh
set -eu
cd "$(dirname "$0")/.."
if [ ! -f backend/.env ]; then cp backend/.env.example backend/.env; fi
if [ ! -f frontend/.env.local ]; then cp frontend/.env.example frontend/.env.local; fi
cd frontend
npm ci
