# vrc_panoprama_picture_manage

requirements redis.

```bash
docker run -e "REDIS_HOST=172.17.0.1" bootjp/vrc_panoprama_picture_manage:latest
```

## how to use

### running application.

```bash
docker pull ghcr.io/bootjp/vrc_panorama_picture_manage:latest
docker run -p 1323:1323 -e "REDIS_HOST=172.17.0.1" ghcr.io/bootjp/vrc_panorama_picture_manage:latest
# print stdout
current temporary token a4df3e9f-a6dc-4967-ac15-02750314f795
# copy temporary token.
```

### setup redirect content.

```bash
curl -X PUT -v -H "Content-Type: application/json" -H "Authorization: Bearer a4df3e9f-a6dc-4967-ac15-02750314f795" http://localhost:1323/api/aaaa -d '{"url":"https://avatars3.githubusercontent.com/u/1306365?v=4"}'
# save to content

# check for content redirect.
curl -v http://localhost:1323/v1/aaaa
# check for content mp4
curl -v http://localhost:1323/v2/aaaa
# get content keys 
curl http://localhost:1323/api/keys
["aaaa"]


```

## Redis settings

`REDIS_HOST` selects the Redis-compatible server. `REDIS_KEY_PREFIX` is optional
and defaults to an empty string for existing installations. Set it to a unique
value such as `panorama:` when using a shared Redis-compatible cluster. It is
applied to the `keys` and `tokens` sets and each panorama URL key; the members
of those sets keep their original values. Existing data must be copied into
the prefixed keys before changing this setting.

`PANORAMA_LOG_TEMP_TOKEN=false` suppresses the temporary admin token in logs.
The default keeps the existing development behavior. When log output is
disabled, provision an admin token in the prefixed `tokens` set before starting
the application.
