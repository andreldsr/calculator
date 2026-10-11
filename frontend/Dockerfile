# syntax=docker/dockerfile:1

# Build: install from the lockfile, then typecheck and bundle.
FROM node:24-alpine AS build
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
RUN npm run build

# Runtime: nginx serves the bundle and proxies /api to the backend.
# The unprivileged image runs as a non-root user and listens on 8080.
FROM nginxinc/nginx-unprivileged:1.30-alpine AS runtime
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 8080
