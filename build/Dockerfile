# images
ARG BUILDER_IMAGE=golang:buster
ARG DISTROLESS_IMAGE=gcr.io/distroless/static

FROM ${BUILDER_IMAGE} AS traebeler-build

# install dependencies
RUN update-ca-certificates \
    && apt install curl \
    && curl -sf https://gobinaries.com/tj/robo | sh

WORKDIR /traebeler

COPY robo.yml .
COPY internal/ internal/
COPY build/version build/version

RUN robo build linux

# copy the binary into the target production image
FROM ${DISTROLESS_IMAGE}
COPY --from=traebeler-build /traebeler/build/dist/traebeler /app/
ENTRYPOINT ["/app/traebeler"]