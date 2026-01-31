# Build stage
FROM node:22-alpine AS builder

# Install pnpm
RUN corepack enable && corepack prepare pnpm@9.15.0 --activate

WORKDIR /app

# Copy workspace configuration
COPY package.json pnpm-workspace.yaml pnpm-lock.yaml* ./

# Copy package files
COPY packages/api-client/package.json ./packages/api-client/
COPY clients/web/package.json ./clients/web/

# Install dependencies
RUN pnpm install --frozen-lockfile

# Copy source files
COPY packages/api-client ./packages/api-client
COPY clients/web ./clients/web

# Copy OpenAPI spec for type generation
COPY api/openapi.yaml ./api/

# Generate types and build
RUN pnpm --filter @feather/api-client generate && \
    pnpm --filter @feather/web build

# Runtime stage - serve static files
FROM nginx:alpine

# Copy built files
COPY --from=builder /app/clients/web/dist /usr/share/nginx/html

# Copy nginx config
COPY clients/web/nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 3000

CMD ["nginx", "-g", "daemon off;"]
