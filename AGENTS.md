# Repository Guidelines

## Project Structure & Module Organization
Backend logic lives in `backend/app/`, with routers in `api/`, business logic in `services/`, shared schemas under `models/`, and helpers in `utils/`. Environment-specific dependencies sit in the various `backend/requirements-*.txt` files. The React interface is under `frontend/src/`, split into `components/`, `hooks/`, `api/`, and `utils/`. Visual assets are stored in `assets/`; model checkpoints and caches belong in `models/` (keep large downloads out of version control). Architectural references live in `docs/architecture.md`, `docs/vllm-direct/implementation-summary.md`, and the rest of `docs/`. The `third_party/` directory is vendored code—open issues before modifying it.

## Build, Test, and Development Commands
Use Docker for an end-to-end stack: `docker compose up --build` starts the Transformers pipeline, while `docker compose -f docker-compose.vllm.yml up --build` and `docker compose -f docker-compose.vllm-direct.yml up --build` target the vLLM variants. For local backend work, create a virtualenv and install dependencies with `python -m venv .venv && source .venv/bin/activate && pip install -r backend/requirements-transformers.txt`, then run `uvicorn backend.app.main:app --reload`. Frontend development uses pnpm: `npm install -g pnpm`, `pnpm install`, and `pnpm run dev`; build output with `pnpm run build`.

## Coding Style & Naming Conventions
Follow PEP 8 in the backend: 4-space indentation, snake_case modules, PascalCase models, and type hints on public functions. Keep HTTP handlers thin—delegate to `services/` for heavy lifting and centralise configuration in `config.py`. In the frontend, prefer two-space indentation, PascalCase component files, `useX` naming for hooks, and colocate feature-specific assets under the matching folder. Enforce lint rules with `pnpm exec eslint src --max-warnings=0` and keep Tailwind utility classes readable (group by layout → typography → state).

## Testing Guidelines
Automated coverage is light; new backend features should ship with `pytest` suites under `backend/tests/` using fast fixtures that stub model loading. Target critical OCR flows first (plain, describe, find, freeform). For UI behaviour, add Vitest + React Testing Library specs in `frontend/src/__tests__/` and run `pnpm exec vitest --run`. Document any manual GPU/vLLM smoke checks in PR descriptions until CI is in place.

## Commit & Pull Request Guidelines
Commits use concise sentence-case summaries (e.g., `Update README.md with new content`) and may reference issues inline `(#12)`. Bundle related work only and update docs/config snapshots alongside code changes. Pull requests need a problem statement, implementation notes, and validation steps (lint, tests, engine mode exercised). Attach screenshots or GIFs for UI tweaks and call out configuration changes (e.g., new `.env` keys). Keep the checklist updated so reviewers know what was verified.
