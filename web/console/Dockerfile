FROM node:22-alpine AS deps
WORKDIR /app
RUN corepack enable
COPY package.json pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

FROM node:22-alpine AS builder
WORKDIR /app
RUN corepack enable
COPY --from=deps /app/node_modules ./node_modules
COPY . .
ENV NEXT_TELEMETRY_DISABLED=1
RUN pnpm build

FROM node:22-alpine AS runner
WORKDIR /app
ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1
ENV PORT=7780
RUN addgroup -S crux && adduser -S crux -G crux
COPY --from=builder --chown=crux:crux /app/public ./public
COPY --from=builder --chown=crux:crux /app/.next/standalone ./
COPY --from=builder --chown=crux:crux /app/.next/static ./.next/static
USER crux
EXPOSE 7780
CMD ["node", "server.js"]
