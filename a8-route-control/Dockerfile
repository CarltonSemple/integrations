FROM alpine:3.3
LABEL works.weave.role=system
COPY ./a8plugin /usr/bin/a8plugin
RUN apk add --no-cache curl
RUN mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2
ENV A8_CONTROLLER_URL http://localhost:31200
ENV A8_REGISTRY_URL http://localhost:31300
ENTRYPOINT ["/usr/bin/a8plugin"]
EXPOSE 5000