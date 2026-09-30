# syntax=docker/dockerfile:1@sha256:ecfaec9ed6d810b56388c508f4121597bfbba70d41a6dfeee4d8cad5f295fc32
FROM --platform=$BUILDPLATFORM node:24.21.0-alpine@sha256:ebfe2f90462722a7a4de65e91990e97fe0d401c70e0e762c5b53302f905ec1c1 AS frontendbuilder

WORKDIR /build

ENV PNPM_CACHE_FOLDER=.cache/pnpm/
ENV PUPPETEER_SKIP_DOWNLOAD=true
ENV CYPRESS_INSTALL_BINARY=0

COPY frontend/pnpm-lock.yaml frontend/package.json frontend/pnpm-workspace.yaml ./
RUN npm install -g corepack && corepack enable && \
    pnpm install --frozen-lockfile
COPY frontend/ ./
ARG RELEASE_VERSION=dev
RUN echo "{\"VERSION\": \"${RELEASE_VERSION/-g/-}\"}" > src/version.json && pnpm run build

FROM --platform=$BUILDPLATFORM ghcr.io/techknowlogick/xgo:go-1.27.x@sha256:8cc742b41f043a4fd45d2f63f1fcd12cb27949342df09efb5561f2aadfbe6da3 AS apibuilder

RUN go install github.com/magefile/mage@latest && \
    mv /go/bin/mage /usr/local/go/bin

WORKDIR /go/src/github.com/MBeggiato/fenster
COPY . ./
COPY --from=frontendbuilder /build/dist ./frontend/dist

ARG TARGETOS TARGETARCH TARGETVARIANT RELEASE_VERSION
ENV RELEASE_VERSION=$RELEASE_VERSION

RUN export PATH=$PATH:$GOPATH/bin && \
	mage build:clean && \
    (cd build && mage release:xgo vikunja "${TARGETOS}/${TARGETARCH}/${TARGETVARIANT}")

RUN mkdir -p /tmp /app/fenster/files && chmod 1777 /tmp

#  ┬─┐┬ ┐┌┐┐┌┐┐┬─┐┬─┐
#  │┬┘│ │││││││├─ │┬┘
#  ┘└┘┘─┘┘└┘┘└┘┴─┘┘└┘

# The actual image
FROM scratch

LABEL org.opencontainers.image.authors='Marcel Beggiato'
LABEL org.opencontainers.image.url='https://github.com/MBeggiato/fenster'
LABEL org.opencontainers.image.documentation='https://github.com/MBeggiato/fenster#readme'
LABEL org.opencontainers.image.source='https://github.com/MBeggiato/fenster'
LABEL org.opencontainers.image.licenses='AGPL-3.0-or-later'
LABEL org.opencontainers.image.title='Fenster'
LABEL org.opencontainers.image.description='Fenster, a fork of Vikunja (https://github.com/go-vikunja/vikunja)'

WORKDIR /app/fenster
ENTRYPOINT [ "/app/fenster/fenster" ]
EXPOSE 3456

COPY --from=apibuilder --chown=1000:1000 --chmod=1777 /tmp /tmp
COPY --from=apibuilder --chown=1000:1000 /app/fenster /app/fenster

USER 1000

ENV FENSTER_SERVICE_ROOTPATH=/app/fenster/
ENV FENSTER_DATABASE_PATH=/db/fenster.db

COPY --from=apibuilder /build/fenster-* fenster
COPY --from=apibuilder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
