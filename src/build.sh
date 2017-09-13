rm -f jobservice

go build

docker build -t harbor.enncloud.cn/harbor/harbor-jobservice:v1.1.1 .

docker push harbor.enncloud.cn/harbor/harbor-jobservice:v1.1.1
