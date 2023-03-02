FROM gcr.io/distroless/static-debian11:nonroot
ENTRYPOINT ["/baton-duo"]
COPY baton-duo /