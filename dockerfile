FROM golang:1.26.6-alpine

WORKDIR /app

COPY go.* ./

RUN go mod download

COPY . .

RUN go build -o main ./cmd/api/main.go

EXPOSE 8081

CMD [ "./main" ]