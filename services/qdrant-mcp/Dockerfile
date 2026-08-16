FROM node:22-slim

RUN apt-get update && apt-get install -y \
    git \
    python3 \
    python3-pip \
    make \
    g++ \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

ARG QDRANT_MCP_VERSION=v3.3.5
RUN git clone --branch ${QDRANT_MCP_VERSION} --depth 1 \
      https://github.com/mhalder/qdrant-mcp-server.git . && \
    npm ci && \
    npm run build

ENV TRANSPORT_MODE=http \
    HTTP_PORT=3000 \
    QDRANT_URL=http://qdrant:6333 \
    EMBEDDING_PROVIDER=ollama \
    EMBEDDING_MODEL=nomic-embed-text \
    OLLAMA_URL=http://ollama:11434 \
    EMBEDDING_BASE_URL=http://ollama:11434

EXPOSE 3000

CMD ["node", "build/index.js"]
