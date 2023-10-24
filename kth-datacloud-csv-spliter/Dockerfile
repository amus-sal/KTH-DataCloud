

# compiler image
FROM golang:1.19.1-alpine3.15 AS build-env

# add git to fetch private go dependencies.
RUN apk update && apk add --no-cache git

WORKDIR /app

# download dependencies.
COPY go.mod go.sum ./
RUN go mod download

# add all files to image.
ADD . ./

# disable cgo to build on alpine.
ENV CGO_ENABLED 0



# build the binary.
RUN GOOS=linux GOARCH=amd64 go build -a -o csvSpliter ./cmd/server/main.go

# create final application image.
FROM alpine:3.15.4
WORKDIR /app
COPY --from=build-env /app/csvSpliter .
COPY --from=build-env /app/config.json .
ENTRYPOINT ./csvSpliter

