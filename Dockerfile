FROM golang:1.26.5-alpine AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /gocalc ./cmd/gocalc

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /gocalc /gocalc
EXPOSE 10000
ENTRYPOINT ["/gocalc", "serve", "--addr", ":10000"]
