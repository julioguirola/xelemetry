FROM golang:1.26.3-trixie

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o app cmd/api/main.go
RUN go build -o migration cmd/migration/main.go

CMD [ "./app" ]