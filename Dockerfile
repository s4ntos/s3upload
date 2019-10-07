FROM centos:6
WORKDIR /wmrupload 
RUN curl -O https://dl.google.com/go/go1.12.7.linux-amd64.tar.gz
RUN yum -y install git
RUN tar -xzf go1.12.7.linux-amd64.tar.gz
RUN mv go /usr/local/
ENV GOROOT=/usr/local/go
ENV GOPATH=/wmrupload/
ENV PATH=$GOPATH/bin:$GOROOT/bin:$PATH
ADD wmrupload.go /wmrupload/
RUN go get github.com/aws/aws-sdk-go/aws
RUN go get github.com/aws/aws-sdk-go/service/s3
RUN go get github.com/spf13/viper
RUN go get "github.com/aws/aws-sdk-go/aws/credentials"
RUN go get "github.com/fsnotify/fsnotify"
RUN go build wmrupload.go 

#### get the build file
# id=$(sudo docker create wmrupload)
# sudo docker cp $id:/wmrupload/wmrupload - > wmrupload
# sudo docker rm -v $id