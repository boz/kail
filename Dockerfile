FROM alpine:latest as alpine_builder
RUN apk --no-cache add ca-certificates
RUN apk --no-cache add tzdata
ENTRYPOINT ["/kail"]
COPY kail /
CMD [""]
