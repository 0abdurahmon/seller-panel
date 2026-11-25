FROM node:24-alpine AS build
WORKDIR /app
COPY package.json package-lock.json ./
RUN --mount=type=cache,target=/root/.npm npm ci --ignore-scripts
COPY . .
RUN npm run build

FROM nginxinc/nginx-unprivileged:1.27-alpine
# The image writes conf.d/default.conf from this template at start, filling in the variables
# that start with RATE_LIMIT_NGINX_ (see docs/RATE_LIMITS.md).
COPY nginx.conf.template /etc/nginx/templates/default.conf.template
ENV NGINX_ENVSUBST_FILTER=^RATE_LIMIT_NGINX_ \
    RATE_LIMIT_NGINX_API_RATE=10r/s \
    RATE_LIMIT_NGINX_API_BURST=30 \
    RATE_LIMIT_NGINX_AUTH_RATE=10r/m \
    RATE_LIMIT_NGINX_AUTH_BURST=5 \
    RATE_LIMIT_NGINX_FILES_RATE=50r/s \
    RATE_LIMIT_NGINX_FILES_BURST=100
COPY security-headers.conf /etc/nginx/snippets/security-headers.conf
COPY spec-proxy.conf /etc/nginx/snippets/spec-proxy.conf
COPY --from=build /app/dist /usr/share/nginx/html
COPY docs/ /usr/share/nginx/html/docs/
COPY --from=build /app/node_modules/swagger-ui-dist/swagger-ui.css /app/node_modules/swagger-ui-dist/swagger-ui-bundle.js /usr/share/nginx/html/docs/
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=5 \
    CMD wget -q --spider http://127.0.0.1:8080/ || exit 1
