# ---- Stage 1: build ----
FROM golang:1.26-alpine AS build
WORKDIR /src

# Copy just the module files first and download deps.
COPY go.mod go.sum ./
RUN go mod download

# Now copy the rest of the source and compile a static binary.
COPY . .
RUN CGO_ENABLED=0 go build -o /server ./cmd/server

# ---- Stage 2: runtime ----
FROM alpine:3.20
COPY --from=build /server /server
EXPOSE 8080
ENTRYPOINT ["/server"]
