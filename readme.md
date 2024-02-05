```bash
docker buld -t cetak_copilot .
docker run -it --rm -p 80:80 -v ${PWD}:/app cetak_copilot sh
```