# API specs

OpenAPI files live here. Swagger UI shows `openapi.yaml` at http://localhost:3000/docs/ when the stack runs with Docker Compose (login: see the Swagger section of the root README).

The folder is mounted read-only into the frontend container, so a saved change shows after a browser refresh, no rebuild needed.

The same spec is served as JSON at http://localhost:3000/swagger (and `/docs/spec/openapi.json`) with the same login. The backend converts `openapi.yaml` on request and notices an edit at once, no restart needed.
