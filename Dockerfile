FROM golang:1.21.0 AS builder
COPY . /workdir
WORKDIR /workdir

RUN go env -w GOPROXY="https://goproxy.cn,direct"
RUN CGO_ENABLED=0 go build -o /bin/app ./main.go

FROM scratch as runner
WORKDIR /app
COPY --from=builder /bin/app .
COPY --from=builder /workdir/index.html .
COPY --from=builder /workdir/file.html .
CMD ["/app/app"]
