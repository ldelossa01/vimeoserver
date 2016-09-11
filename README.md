# Vimeoserver

##### VimeoServer's main focus is to act as a byte-range proxy. The user may create http requests providing two url parameters:

s: source URL
range: Byte range

A typical curl request to the server would look like the following:

```
curl 'localhost:8000/?s="http://storage.googleapis.com/vimeo-test/work-at-vimeo.mp4"&range=600-1000'
```

## Features:

VimeoServer implements a LRU in-memory cache. The cache implements an interface defined in cache.go. You are free to implement your own cache into the server as long as you adhere to the interface.

The cache itself uses a min-heap to track epoch time of byte entries. Each eviction cycle pop's the object with the lowest epoch timestamp and evicts this object from cache.

VimeoServer will also act as a simple proxy, however the source URL will require the Accept-Ranges header and the bytes value for this header declared. If these requirements are met VimeoServer will proxy the request with no need for byte ranges or partial responses.

Tests are provided which confirm the functionality of the server along with the cache. These can be ran by:

```
cd ./vimeoserver/server
go test
```
