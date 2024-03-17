```bash
docker build -t cetak_copilot .
docker run -it --rm -p 80:80 -v ${PWD}:/code cetak_copilot sh
air -c .air.toml
```

## Authentication Issue Resolve
https://github.com/googleworkspace/go-samples/issues/76#issuecomment-1304902886