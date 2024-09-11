FROM golang:1.23.1 AS builder
COPY . /workdir
WORKDIR /workdir

RUN go env -w GOPROXY="https://goproxy.cn,direct"
RUN CGO_ENABLED=0 go build -o /bin/app ./cmd/main.go

FROM scratch as runner
WORKDIR /app
COPY --from=builder /bin/app .
CMD ["/app/app"]
