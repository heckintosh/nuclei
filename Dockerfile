FROM golang:1.18.4-alpine as build-env
RUN go install -v github.com/heckintosh/nuclei/v2/cmd/nuclei@latest

FROM alpine:3.16.0
RUN apk add --no-cache bind-tools ca-certificates chromium
COPY --from=build-env /go/bin/nuclei /usr/local/bin/nuclei
ENTRYPOINT ["nuclei"]
