FROM golang:1.21 AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 go build -o main .

FROM scratch
COPY --from=builder /app/main /main
CMD ["/main"]
