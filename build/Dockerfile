FROM golang:1.15-alpine AS traebeler-build

# install dependencies
RUN apk add git curl
RUN curl -sf https://gobinaries.com/tj/robo | sh

WORKDIR /traebeler

COPY robo.yml .
COPY internal/ internal/
COPY build/version build/version

RUN robo build

FROM golang:1.15-alpine
COPY --from=traebeler-build /traebeler/build/dist/traebeler /app/
RUN ["chmod", "a+x", "/app/traebeler"]
CMD ["/app/traebeler"]