# Builder separetes the environment for running to reduce latency.
FROM golang:1.25 as builder
# Decides where to run the files.
WORKDIR /app
# Copies the given files to the working directory.
COPY go.mod go.sum ./
# runs commands in the shell of the base image. ()
RUN go mod download
# Copies all files in the current directory.
COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o server .

# Second stage.
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=builder /app/server .
CMD ["./server"]
