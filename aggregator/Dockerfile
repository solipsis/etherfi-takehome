FROM golang:1.19-alpine

WORKDIR /app

COPY go.mod ./
COPY main.go ./

RUN go build -o aggregator .

EXPOSE 8085

CMD [ "./aggregator" ]
