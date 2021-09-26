## Miniblog web server

[OpenAPI specification.](api.yaml)

Run server:
```bash
$ go run server.go
```

Run server from docker container:
```bash
$ docker build . -t miniblog
$ docker run -e SERVER_PORT=8080 -p 8080:8080 miniblog
```