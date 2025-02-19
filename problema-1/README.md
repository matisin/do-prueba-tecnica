## Como lanzar la api
Para correr la api baste con levanta el docker compose, cambiar el puerto si esta ocupado: 

```sh
docker compose up
```

## Test
Los test se corren en la consola en go por modulo con los siguientes comandos:

### test unitarios
```sh
go test -v ./internal/app
```

### test e2e

```sh
go test -fuzz=. -fuzztime=2s -v ./internal/infra/http
```
