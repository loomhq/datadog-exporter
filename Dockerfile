FROM scratch
COPY datadog-exporter /
ENTRYPOINT ["/datadog-exporter"]
