```bash
docker build -t cetak_copilot .
docker run -it --rm -p 80:80 -v ${PWD}:/code cetak_copilot sh
air -c .air.toml
```