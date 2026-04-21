FROM node:20-alpine AS deps

WORKDIR /app

RUN corepack enable pnpm

COPY package.json pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

FROM deps AS dev

WORKDIR /app

COPY index.html ./
COPY package.json pnpm-lock.yaml tsconfig.json tsconfig.node.json vite.config.ts ./
COPY src ./src

EXPOSE 5173

CMD ["pnpm", "dev", "--host", "0.0.0.0", "--port", "5173"]
