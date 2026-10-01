FROM oven/bun:1.4.2

WORKDIR /app
COPY package.json bun.lock ./
RUN bun install --frozen-lockfile --production
COPY src ./src

CMD ["bun", "src/index.ts"]
