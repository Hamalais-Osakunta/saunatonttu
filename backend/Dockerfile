FROM docker.io/golang:latest
RUN echo > /empty
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bt-telegram

FROM scratch
COPY --from=0 /empty /.env
COPY --from=0 /bt-telegram /bt-telegram
ENTRYPOINT ["/bt-telegram"]
