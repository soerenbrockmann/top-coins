FROM golang:1.14

RUN mkdir /app
ADD . /app
WORKDIR /app
RUN go mod init github.com/coding-challenge/pricing-service
RUN go build -o main .
CMD ["/app/main"]