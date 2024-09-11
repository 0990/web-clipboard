## docker安装

```dockerfile
version: '3'

services:
  chinadns:
    image: 0990/web-clipboard:latest
    container_name: web-clipboard
    ports:
      - "80:80"
```