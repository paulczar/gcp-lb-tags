FROM golang:1.9.2

WORKDIR /go/src/app
COPY . .
RUN go-wrapper download
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /app/gcp-lb-tags .

FROM ubuntu:latest
RUN apt-get update && apt-get install -yq curl
WORKDIR /app
COPY --from=0 /app /app
COPY pks.sh /app/pks.sh

CMD ["/app/gcp-lb-tags", "create"]
