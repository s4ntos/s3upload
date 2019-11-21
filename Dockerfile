
FROM golang:1.10.0-alpine3.7

WORKDIR /s3upload 

## build
ADD wmrupload.go /s3upload/
RUN go get github.com/aws/aws-sdk-go/aws
RUN go get github.com/aws/aws-sdk-go/service/s3
RUN go get github.com/spf13/viper
RUN go get "github.com/aws/aws-sdk-go/aws/credentials"
RUN go get "github.com/fsnotify/fsnotify"
RUN go build s3upload.go 

RUN apk update && apk upgrade && \
    apk add --no-cache git

RUN go build -o s3upload s3upload.go



#### get the build file
# id=$(sudo docker create s3upload)
# sudo docker cp $id:/s3upload/s3upload - > s3upload
# sudo docker rm -v $id