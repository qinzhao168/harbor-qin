FROM vmware/harbor-jobservice:v1.1.1

RUN mkdir -p /usr/local/go/lib/time/

ADD zoneinfo.zip /usr/local/go/lib/time/

ADD jobservice /harbor/harbor_jobservice