# syntax=docker/dockerfile:1

FROM golang:1.16.8-alpine
RUN apk add gcc libc-dev
WORKDIR /app
COPY go.mod ./
COPY go.sum ./
RUN go mod download
COPY *.go ./
COPY *.css ./
COPY *.html ./
RUN go build -o /thatthing
EXPOSE 8080
ENV pass ""
ENV name ""
ENV db "pg"
CMD ["sh", "-c", "/thatthing -pass ${pass} -name ${name}"]
