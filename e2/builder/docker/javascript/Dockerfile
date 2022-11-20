FROM ghcr.io/suborbital/javy:v0.3.0 as javy
FROM suborbital/subo:dev as subo

FROM node:16-bullseye-slim
WORKDIR /root/runnable
# Propagate our root permissions for our home folder to everyone. This allows
# npm scripts (which get run as whatever user owns the mounted runnable
# directory) to access common home folder resources (caches, etc.).
RUN mkdir /root/suborbital && chmod -R o=u /root

COPY --from=javy /usr/local/bin/javy /usr/local/bin
COPY --from=subo /go/bin/subo /usr/local/bin
