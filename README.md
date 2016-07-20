# Exchange Rates Converter

This is a simple HTTP API that wraps http://fixer.io/ with extra functionality.

## Installation

Using go tooling:

`go get github.com/viru/xrate`

or checkout this repository and build a Docker image:

`docker build -t xrate:0.1 .`

## Running

Try `xrate -h` for supported flags, but in general you can omit them:

```sh
$ xrate
2016/07/19 23:49:31 Starting HTTP listener on :8080
```

or if you've build the Docker image:

```sh
$ docker run -p8080:8080 -it --rm --name xrate xrate:0.1
+ exec app
2016/07/19 21:55:25 Starting HTTP listener on :8080
```

## Usage

```sh
$ curl 'http://localhost:8080/convert?amount=200&currency=SEK'
{"amount":200,"currency":"SEK","converted":{"AUD":31.03,"BGN":41.21,...}}

$ curl -H Accept:application/xml 'http://localhost:8080/convert?amount=200&currency=USD'
<?xml version="1.0" encoding="UTF-8"?>
<Rates><Amount>200</Amount><Currency>USD</Currency><Converted><KRW>227920</KRW>...</Rates>
```