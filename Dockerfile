FROM alpine:3.17.3 as alpine_builder
RUN apk --no-cache add ca-certificates
RUN apk --no-cache add tzdata
ENTRYPOINT ["/kail"]
COPY kail /
CMD [""]
