# vrc_panoprama_picture_manage

requirements redis.

```bash
docker run -e "REDIS_HOST=172.17.0.1" bootjp/vrc_panoprama_picture_manage:latest
```

## how to use 


### running application. 

```bash

docker pull bootjp/vrc_panoprama_picture_manage:latest
docker run  -p 1080:1323 -e "REDIS_HOST=172.17.0.1" bootjp/vrc_panoprama_picture_manage:latest
# print stdout
current temporary token a4df3e9f-a6dc-4967-ac15-02750314f795
# copy temporary token. 
```

### setup redirect content.
```bash
curl -X POST http://somehost:1323/api/update -H 'Content-Type: application/json' -d '{"key":"keystring", "token":"a4df3e9f-a6dc-4967-ac15-02750314f795", "URL":"https://avatars3.githubusercontent.com/u/1306365?v=4"}'
# save to content

# check for content redirect.
curl -v http://somehost:1323/r/keystring 

### example output
*   Trying ::1...
* TCP_NODELAY set
* Connected to localhost (::1) port 1323 (#0)
> GET /r/keystring HTTP/1.1
> Host: localhost:1323
> User-Agent: curl/7.64.1
> Accept: */*
>
< HTTP/1.1 302 Found
< Cache-Control: no-store
< Location: https://avatars3.githubusercontent.com/u/1306365?v=4
< Date: Sat, 19 Oct 2019 07:42:25 GMT
< Content-Length: 0
<
* Connection #0 to host localhost left intact
* Closing connection 0

```
