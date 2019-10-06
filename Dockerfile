cd /tmp 
wget https://dl.google.com/go/go1.12.7.linux-amd64.tar.gz
tar -xzf go1.12.7.linux-amd64.tar.gz
mv go /usr/local/
mkdir -p /tmp/wmrupload/
export GOROOT=/usr/local/go
export GOPATH=/tmp/wmrupload/
export PATH=$GOPATH/bin:$GOROOT/bin:$PATH
go get github.com/aws/aws-sdk-go/aws
go get github.com/aws/aws-sdk-go/service/s3
go get github.com/spf13/viper
go get github.com/radovskyb/watcher