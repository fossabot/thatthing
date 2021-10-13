FROM golang:1.16.8-alpine3
ADD . /
WORKDIR /app
RUN go build -o main .
CMD ["/main"]
