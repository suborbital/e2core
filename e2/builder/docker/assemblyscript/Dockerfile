FROM suborbital/subo:dev as subo

FROM node:16-buster-slim
WORKDIR /root/runnable
COPY --from=subo /go/bin/subo /usr/local/bin
RUN npm install -g npm@latest
