# ipecho.dev

[![CI](https://github.com/jacob-lineberry/ipecho.dev/actions/workflows/ci.yml/badge.svg)](https://github.com/jacob-lineberry/ipecho.dev/actions/workflows/ci.yml)

Your IP over HTTP. A small Go service that returns your public IP address as plain text or JSON.

## Usage

```console
$ curl https://ipecho.dev
203.0.113.42

$ curl https://ipecho.dev/json
{"ip":"203.0.113.42"}
```

Use `curl -4` or `curl -6` to ask for your IPv4 or IPv6 address.

| Endpoint      | Response                                 |
| ------------- | ---------------------------------------- |
| `GET /`       | Your IP as `text/plain`                  |
| `GET /json`   | `{"ip": "..."}` as `application/json`    |
| `GET /health` | `ok`, for health checks                  |

`/` and `/json` allow 120 requests per minute per IP and answer `429 Too Many Requests` past that.

## Development

Requires Go (the version is in `go.mod`) and [golangci-lint](https://golangci-lint.run/) for `make check`.

```sh
make check            # format, tidy, vet, lint and test, as CI runs it
go run ./cmd/server   # serves on :8080, or on $PORT
```

Or with Docker:

```sh
docker build -t ipecho .
docker run --rm -p 8080:8080 ipecho
```

## License

[MIT](LICENSE)
