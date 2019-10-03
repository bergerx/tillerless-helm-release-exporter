FROM golang:1.13.0-stretch AS builder
WORKDIR $GOPATH/src/github.com/bergerx/tillerless-helm-release-exporter
COPY . .
RUN  CGO_ENABLED=0 GOOS=linux go build -a -installsuffix nocgo -o /tillerless-helm-release-exporter

FROM scratch
COPY --from=builder /tillerless-helm-release-exporter /tillerless-helm-release-exporter
ENTRYPOINT ["/tillerless-helm-release-exporter"]
EXPOSE 8080
ARG BUILD_DATE
ARG VCS_REF
LABEL org.label-schema.build-date=$BUILD_DATE \
      org.label-schema.vcs-ref=$VCS_REF \
      org.label-schema.vcs-url="https://github.com/bergerx/tillerless-helm-release-exporter"
