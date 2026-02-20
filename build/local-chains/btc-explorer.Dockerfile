FROM node:18-alpine

RUN apk add --no-cache python3 make g++ zeromq-dev

RUN npm install -g btc-rpc-explorer@3.4.0

ENV BTCEXP_HOST=0.0.0.0
ENV BTCEXP_PORT=3002

EXPOSE 3002

CMD ["btc-rpc-explorer"]
