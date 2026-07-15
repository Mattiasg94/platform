# The orchestrator as a long-lived Cloud Run service. Its own image, unrelated as
# a file to the agent's — and it bundles no agent build context yet. That cargo
# only becomes necessary when the agent build moves to Cloud Build (a later slice),
# because that is when the orchestrator becomes the thing that submits the build.
FROM golang:1.25-alpine AS build
WORKDIR /src
# Download deps first so an edit that doesn't touch go.mod reuses the layer.
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /out/orchestrator ./cmd/orchestrator

FROM alpine:3.21
# git: the run pipeline shells out to it. ca-certificates: HTTPS to Google and
# GitHub. Neither is exercised by the stubbed trigger, but the image should be
# whole rather than surprise us the day the pipeline is un-stubbed.
RUN apk add --no-cache git ca-certificates
COPY --from=build /out/orchestrator /usr/local/bin/orchestrator
# Cloud Run injects PORT and routes to it; 8080 is the fallback the server uses.
ENV PORT=8080
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/orchestrator"]
