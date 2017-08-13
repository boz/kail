FROM alpine

ADD ./ktail-linux /ktail

ENTRYPOINT ./ktail
