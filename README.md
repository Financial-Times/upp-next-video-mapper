[![CircleCI](https://circleci.com/gh/Financial-Times/upp-next-video-mapper.svg?style=svg)](https://circleci.com/gh/Financial-Times/upp-next-video-mapper)
[![Coverage Status](https://coveralls.io/repos/github/Financial-Times/next-video-mapper/badge.svg?branch=master)](https://coveralls.io/github/Financial-Times/next-video-mapper?branch=master)

# Next Video Mapper
Next Video Mapper is responsible for listening for new video events, then transforming it to a structure amenable to processing by UPP and putting this structure back on the queue.

We currently support videos created and published in Next Video Editor, that get into our stack through Cms-Notifier. In order to be processed by Next Video Mapper, a video content *MUST* include the header `X-Origin-Id=next-video-editor` .

Next Video Mapper is able to process messages for both publish and un-publish events.

## Installation
In order to install, execute the following steps:
```
go get -u github.com/Financial-Times/upp-next-video-mapper
cd $GOPATH/src/github.com/Financial-Times/upp-next-video-mapper
go build -mod=readonly .
```

## Running Locally

```
export Q_ADDR="http://172.23.53.64:8080" ## you'll have to change the address to your queue
export Q_GROUP=nextVideoMapper
export Q_READ_TOPIC=NativeCmsPublicationEvents
export Q_READ_QUEUE=kafka
export Q_WRITE_TOPIC=CmsPublicationEvents
export Q_WRITE_QUEUE=kafka
export Q_AUTHORIZATION=$(etcdctl get /ft/_credentials/kafka-bridge/authorization_key) ## this is not exact, you'll have to get it from the cluster's etcd
export PORT=... ## 8080 by default, only necessary when you need a custom port for running locally 
go build .
./upp-next-video-mapper
```

## Running Test

```
go test -mod=readonly -race ./...
```

## Expected behaviour 

A transformation from a native video JSON to a UPP video content. The event is triggered when a video will be published/deleted from Next Video Editor.

The mapper reads events from the NativeCmsPublicationEvents topic and sends the transformed package to the CmsPublicationEvents topic. 

### Publish event

An example of native video JSON model can be found [here](https://gist.github.com/tosan88/580a10da0b5ef3df0a89d40acfe957c7) 

You should receive a response body like:

```json
{
    "contentUri": "http://next-video-mapper.svc.ft.com/video/model/e2290d14-7e80-4db8-a715-949da4de9a07",
    "payload": {
        "uuid": "e2290d14-7e80-4db8-a715-949da4de9a07",
        "title": "Trump trade under scrutiny",
        "standfirst": "Mike Mackenzie provides analysis of the morning's market news",
        "description": "Global equities are on the defensive, led by weaker commodities and financials as investors scrutinise the viability of the Trump trade. The FT's Mike Mackenzie reports.",
        "byline": "Filmed by Niclola Stansfield. Produced by Seb Morton-Clark.",
        "identifiers": [{
            "authority": "http://api.ft.com/system/NEXT-VIDEO-EDITOR",
            "identifierValue": "e2290d14-7e80-4db8-a715-949da4de9a07"
        }],
        "brands": [{
            "id": "http://api.ft.com/things/dbb0bdae-1f0c-11e4-b0cb-b2227cce2b54"
        }],
        "mainImage": "ffc60243-2b77-439a-38af-98acd99af4ca",
        "transcript": "<p>Here's what we're watching with trading underway in London. Global equities under pressure led by weaker commodities and financials as investors scrutinise the viability of the Trump trade. The dollar is weaker. Havens like yen, gold, and government bonds finding buyers. </p><p>As the dust settles over the failure to replace Obamacare, focus now on whether tax reform and other fiscal measures will eventuate. This is where the rubber meets the road for the Trump trade. High flying equity markets had been underpinned by the promise of big tax cuts and fiscal stimulus. And Wall Street is souring. </p><p>One big beneficiary of lower corporate taxes under Trump are small caps. They are now down 2 and 1/2% for the year. While the sector is still much higher since November, this is a key market barometer of prospects for the Trump trade. </p><p>Now while many still think some measure of tax reform or spending will eventuate, markets are very wary, namely of the risk that Congress and the Trump administration fail to reach agreement on legislation, that unlike health care reform, matters a great deal more to investors. </p><p>[MUSIC PLAYING] </p>",
        "captions": [{
            "url": "https://next-video-editor.ft.com/e2290d14-7e80-4db8-a715-949da4de9a07.vtt",
            "mediaType": "text/vtt"
        }],
        "dataSource": [{
            "binaryUrl": "http://ftvideo.prod.zencoder.outputs.s3.amazonaws.com/e2290d14-7e80-4db8-a715-949da4de9a07/0x0.mp3",
            "mediaType": "audio/mpeg",
            "duration": 65904,
            "audioCodec": "mp3"
        },
        {
            "binaryUrl": "http://ftvideo.prod.zencoder.outputs.s3.amazonaws.com/e2290d14-7e80-4db8-a715-949da4de9a07/640x360.mp4",
            "pixelWidth": 640,
            "pixelHeight": 360,
            "mediaType": "video/mp4",
            "duration": 65940,
            "videoCodec": "h264",
            "audioCodec": "aac"
        },
        {
            "binaryUrl": "http://ftvideo.prod.zencoder.outputs.s3.amazonaws.com/e2290d14-7e80-4db8-a715-949da4de9a07/1280x720.mp4",
            "pixelWidth": 1280,
            "pixelHeight": 720,
            "mediaType": "video/mp4",
            "duration": 65940,
            "videoCodec": "h264",
            "audioCodec": "aac"
        }],
        "canBeDistributed": "yes",
        "canBeSyndicated": "yes",
        "accessLevel": "free",
        "type": "MediaResource",
        "lastModified": "2017-04-13T10:27:32.353Z",
        "publishReference": "tid_123123",
        "webUrl": "https://www.ft.com/content/e2290d14-7e80-4db8-a715-949da4de9a07",
        "canonicalWebUrl": "https://www.ft.com/content/e2290d14-7e80-4db8-a715-949da4de9a07",
        "promotionalTitle": "promoTitleHere",
        "promotionalStandfirst": "promotionalStandfirstHere"
    },
    "lastModified": "2017-04-13T10:27:32.353Z"
}
```

### Un-publish/delete event
The request body should have the following format:
```json
{
    "deleted": true,
    "lastModified": "2017-04-04T14:42:58.920Z",
    "publishReference": "tid_bycjmmcj4r",
    "type": "video",
    "id": "e2290d14-7e80-4db8-a715-949da4de9a07"
}
```

You should receive a response body like:
```json 
{
    "contentUri": "http://next-video-mapper.svc.ft.com/video/model/e2290d14-7e80-4db8-a715-949da4de9a07",
    "payload": {},
    "lastModified": "2017-04-09T05:19:49.756Z"
}
```

## Build and Deployment

### DockerHub

https://hub.docker.com/r/coco/upp-next-video-mapper/builds/
