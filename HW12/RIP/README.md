## Программирование

### RIP

Приложение написано на языке Go.

Для его запуска нужно из корня проекта вызвать

```angular2html
go run ./rip.go
```

Топология сети задается в [json-конфиге](as.json).

### Работа кода для части А

Вывод ```Next Hop``` был добавлен позднее.

![image](../pictures/rip1.png)

### Работа кода для части Б

Вывод ```Next Hop``` был добавлен позднее.

![image](../pictures/rip2.png)

После добавления ```Next Hop```:

![image](../pictures/rip4.png)

### Работа кода для части В

Для распараллеливания я воспользовался горутинами. Синхронизация между ними -- ```sync.WaitGroup``` и ```sync.Map```.

![image](../pictures/rip3.png)