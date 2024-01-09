FROM golang:1.21 AS build-stage
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /gobot

FROM gcr.io/distroless/base-debian12 AS release-stage
WORKDIR /gobot
COPY --from=build-stage /gobot /gobot/gobot
USER nonroot:nonroot
ENTRYPOINT ["/gobot/gobot"]
