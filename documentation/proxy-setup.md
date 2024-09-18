# How to setup a proxy server for your web application

## Traefik Proxy dynamic config

```yaml

http:
  middlewares:
    forward-auth:
      forwardAuth:
        address: "http://auth-service:8080/auth"
        trustForwardHeader: true

    gzip:
      compress: true

    redirect-to-https:
      redirectScheme:
        scheme: https

  routers:
    # HTTP Router - Redirect to HTTPS
    http-router:
      entryPoints:
        - http
      middlewares:
        - redirect-to-https
      rule: Host(`hamis-sauna.lim.fi`) && PathPrefix(`/`)
      service: http-service

    # HTTPS Router
    https-router:
      entryPoints:
        - https
      middlewares:
        - gzip
      rule: Host(`hamis-sauna.lim.fi`) && PathPrefix(`/`)
      service: mirror-service
      tls:
        certResolver: letsencrypt

    # Custom Router for /api/receive-bt
    bt-receive-router:
      entryPoints:
        - https
      rule: Host(`hamis-sauna.lim.fi`) && PathPrefix(`/api/receive-bt`)
      service: mirror-service
      tls:
        certResolver: letsencrypt

  services:
    http-service:
      loadBalancer:
        servers:
          - url: http://mirror-service:80

    mirror-service:
      mirroring:
        service: main-service
        mirrors:
          -
            name: bt-receive-service
            percent: 100
    main-service:
      loadBalancer:
        servers:
          -
            url: 'http://tg-bot-service:80'
    bt-receive-service:
      loadBalancer:
        servers:
          -
            url: 'http://data-ingest-service:8000'
```