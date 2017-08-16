FROM alpine

ADD ./kail-linux /kail

ENTRYPOINT ["./kail"]
CMD [""]
