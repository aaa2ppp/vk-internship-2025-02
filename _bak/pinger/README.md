
Сервис Pinger. Получает список всех docker-контейнеров, пингует их и отправляет данные в базу через API frontend.

От куда получает? Когда получает?

На старте из файла?
Сам сканирует сегмент в котором находится?
От нас хотят реальный пинг icmp 8:Echo Request/0:Echo Reply
Или tcp syn на порт?


Обычно если мне нужно посмотреть, что есть рядом в сегменте,
я делаю:

```sh
ifconfig | grep inet | grep -v inet6 | grep -v 127.0.0.1
ippref=192.168.xxx. # то, что выдал ifconfig
for i in $(seq 1 254); do ping -c1 $ippref$i >/dev/null 2>&1 & done; sleep 1; arp -a | grep -v incomplete
```

Ok
Будем получать или из файла. Имя файла будем передавать, через праметр командной строки -f
Если параметр не задан или имя фйла -, то будем чистать с stdin. каждая строка IP или FQDN и опционально порт
Если задан порт случае будем делать tcp syn ping иначе icmp ping.
Иначе если задан парамет -s будем сканировать соседей по сегменту.

Oops!..
Std неумеет icmp. 
https://github.com/prometheus-community/pro-bing
A simple but powerful ICMP echo (ping) library for Go, inspired by go-ping & go-fastping.

