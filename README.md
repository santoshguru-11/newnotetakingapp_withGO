# newnotetakingapp_withGO

1. Generate SSL certificates if you haven't already:
```bash
openssl req -x509 -newkey rsa:4096 -sha256 -days 365 -nodes \
  -keyout server.key -out server.crt \
  -subj "/CN=localhost" \
  -addext "subjectAltName=DNS:localhost,IP:127.0.0.1,IP:::1"
```

2. Run the server:
```bash
go run noteapp.go
```

3. Visit the application:
```
https://localhost:8444
```

Note: You'll see a security warning in your browser because we're using a self-signed certificate. This is normal for development.