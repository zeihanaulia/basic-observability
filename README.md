# Basic Observability and Monitoring

Learn about the monitoring process, using Grafana, Prometheus, Loki and Jaeger.

## Install docker plugin

```
 docker plugin install grafana/loki-docker-driver:latest --alias loki --grant-all-permissions
```

## Running

```
docker-compose -f docker-compose.yml up --no-deps --build
```

Wait until mysql ready and all service restarted.

```
docker-compose down -v
```

## Link 
- App
    - [x] Payment
    - [x] Order Service
    - [x] Order Worker
    - [ ] Warehouse

- [Prometheus](http://localhost:9000/targets)

## Referensi:

- https://www.observability.blog/nginx-monitoring-with-prometheus/
- https://grafana.com/docs/loki/latest/clients/docker-driver/