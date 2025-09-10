## Marcus

##### _Hello Robert_

### Environment Variables

+ DISCORD_BOT_TOKEN
  + Discord Bot Authentication Token
+ AUDIO_DIR
  + Generated audio files will be cached in this directory
+ magic_key
  + The secret sauce
+ OPEN_ROUTER_KEY
  + An open router API key. While AI commands should still work without this key, they will be rate limited.

### Building

Two Dockerfiles are included

+ Dockerfile-base
  + This file is used to build the base image which includes ffmpeg and Go
  + You'll only need to build this image once
+ Dockerfile
  + Compiles and executes the actual bot
  + Rebuild whenever you update any Go files


### Commands

+ `!marcus <INPUT>`
  + General TTS
+ `!marcus-insult`
  + Speaks a random insult from the Evil insult API
+ `!marcus-cache`
  + Responds with a list of all cached audio files 

### Tips

Marcus will attempt to write all generated audio files to disk so that they do not need to be regenerated.
to make sure these files aren't lost after the container is killed, you should mount a host path into the container.

`docker run -v <HOSTPATH>:/audio [ENV] marcus`