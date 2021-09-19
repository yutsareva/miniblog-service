## Miniblog web server

[OpenAPI specification.](api.yaml)

Run docker container:
```bash
$ docker build . -t miniblog
$ docker run -e SERVER_PORT=8899 -p 8899:8899 miniblog
```
