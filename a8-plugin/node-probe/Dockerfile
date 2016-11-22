FROM alpine:3.3
LABEL works.weave.role=system
COPY ./a8plugin /usr/bin/a8plugin
RUN apk add --no-cache curl
RUN mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2
ENV A8_CONTROLLER_URL http://169.46.71.25:31200
ENV A8_REGISTRY_URL http://169.46.71.25:31300
ENV ES_URL http://169.46.71.25:30200
ENTRYPOINT ["/usr/bin/a8plugin"]
EXPOSE 5000