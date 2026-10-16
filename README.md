# Check deadlocks the connection pool in the weight2 fast path

Reproduction for an OpenFGA v1.22.0 bug with Postgres: on a fresh server with the default configuration, a BatchCheck
of 50 items through a tuple to userset whose parent relation is a union runs into the deadline.

```bash
docker compose up -d --wait
go run .
```

```text
BatchCheck took 3.024s: 0 allowed, 49 errors context deadline exceeded
```

Run `docker compose down -v` before the next run: later BatchChecks on the same server succeed.
