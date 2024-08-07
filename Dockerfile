FROM golang:1.22 AS build

WORKDIR /usr/src/app

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN go build -ldflags "-s -w" -trimpath -v -o /usr/bin/app .

FROM debian:latest

RUN apt update && apt install -y ca-certificates && apt clean

COPY --from=build /usr/bin/app /usr/bin/app

CMD ["app"]
