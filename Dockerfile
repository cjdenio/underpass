FROM golang:1.17-alpine AS builder

WORKDIR /usr/src/app

COPY . .

RUN go build -o server ./cmd/server

FROM alpine:latest AS runner

COPY --from=builder /usr/src/app/server /usr/bin/underpass

CMD [ "underpass" ]