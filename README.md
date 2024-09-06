# web-clipboard
简洁的网页剪切板，方便不同设备分享剪切板或文件

# 安装
## docker
```dockerfile
version: '3'

services:
  chinadns:
    image: 0990/web-clipboard:latest
    container_name: web-clipboard
    ports:
      - "80:80"
```

# 使用
访问 127.0.0.1,拖入文件或ctrl+V,即可上传文件或剪切板<br>
再次访问127.0.0.1,即可查看剪切板内容<br>
![img_2.png](doc/img_2.png)