FROM node:22.23.1-bookworm-slim AS renderer
ENV PUPPETEER_SKIP_DOWNLOAD=true
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci --omit=dev --no-fund --no-audit
COPY scripts/build-assets.mjs scripts/build-assets.mjs
RUN node scripts/build-assets.mjs

FROM renderer AS runtime-deps
# UI assets are embedded by Go; retain only the content compiler's dependencies.
# Declaration files and source maps are unused by the production JS runtime.
RUN rm -rf node_modules/gsap node_modules/lxgw-wenkai-webfont \
    && find node_modules -type f \( -name '*.map' -o -name '*.d.ts' -o -name '*.d.mts' -o -name '*.d.cts' \) -delete

FROM golang:1.27.1-bookworm AS backend
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
COPY --from=renderer /app/internal/blog/web/static ./internal/blog/web/static
COPY --from=renderer /app/internal/blog/web/font-head.html ./internal/blog/web/font-head.html
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /jianxi ./cmd/jianxi \
    && install -d -o 10001 -g 10001 -m 0700 /server-root/data \
    && install -d -m 1777 /server-root/tmp

FROM node:22.23.1-bookworm-slim AS mcp
RUN apt-get update && apt-get install -y --no-install-recommends chromium-headless-shell fontconfig fonts-noto-cjk ca-certificates dumb-init \
    && rm -rf /var/lib/apt/lists/* \
    && rm -f /usr/share/fonts/opentype/noto/NotoSerifCJK-*.ttc && fc-cache -f \
    && groupadd -g 10001 jianxi && useradd -r -u 10001 -g jianxi jianxi && mkdir -p /data /app && chown jianxi:jianxi /data /app
RUN install -d -o jianxi -g jianxi -m 0700 /home/jianxi
WORKDIR /app
COPY --from=backend /jianxi /usr/local/bin/jianxi
COPY --from=runtime-deps /app/node_modules ./node_modules
COPY package.json package-lock.json ./
COPY scripts/compile.mjs ./scripts/compile.mjs
ENV HOME=/home/jianxi PUPPETEER_EXECUTABLE_PATH=/usr/bin/chromium-headless-shell JIANXI_CHROMIUM_HEADLESS_SHELL=1 JIANXI_CHROMIUM_NO_SANDBOX=1 NODE_ENV=production
USER jianxi
ENTRYPOINT ["dumb-init", "--", "jianxi"]
CMD ["mcp", "--renderer", "/app/scripts/compile.mjs"]

# The deployed server is a static Go binary; all heavy compilation runs in MCP.
FROM scratch AS server
COPY --from=backend /jianxi /jianxi
COPY --from=backend /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=backend /server-root/ /
WORKDIR /data
USER 10001:10001
VOLUME /data
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s CMD ["/jianxi", "healthcheck"]
ENTRYPOINT ["/jianxi"]
CMD ["serve", "--data", "/data", "--addr", "0.0.0.0:8080"]
