FROM docker.io/golang:latest
RUN echo > /empty
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bt-telegram

FROM scratch
# Take root SSL certificates from the Go builder image
COPY --from=0 /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=0 /empty /.env
COPY --from=0 /bt-telegram /bt-telegram
ENTRYPOINT ["/bt-telegram"]
