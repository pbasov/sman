FROM golang:1.23.1-alpine AS builder
WORKDIR /app
COPY go.mod ./
COPY main.go ./
RUN go mod tidy
RUN go build -o kube-secret-api
COPY ./frontend ./frontend

FROM alpine:3.14
WORKDIR /root/
COPY --from=builder /app/kube-secret-api .
COPY --from=builder /app/frontend ./frontend
CMD ["./kube-secret-api"]
