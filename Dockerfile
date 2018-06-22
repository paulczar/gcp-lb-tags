FROM golang:1.9.2

WORKDIR /go/src/app
COPY . .
RUN go-wrapper download
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /app/go-lb-tags .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=0 /app /app
CMD ["/app/go-lb-tags"]
