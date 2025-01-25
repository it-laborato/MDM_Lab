Запуск (желательно все сделать с помощью make команд)



1. Скачать зависимости
```sh 
make deps
```

2. Сгенерировать фронт и го файл для него
```sh
make generate
```

3. Собрать билды 
```sh 
make build
```

4. Поднять вспомогательные образы

```sh
docker compose up -d --build
```

5. Накатить миграции (может первые секунд 10 логировать EOF ит окей)
```sh 
./build/mdmlab prepare db --dev
```

6. Запустить сервер
```sh
./build/mdmlab serve --dev
```
./build/mdmlabctl package --type=pkg --enable-scripts --mdmlab-url=https://localhost:8080 --enroll-secret=CgSw8B4BmHMReSzEGRUWBTmY3BpzR142
после выполнения 6 пункта можно перейти на localhost:8080 и там будет страница с регистрацией


(Все команды разом)

```sh 
make deps && make generate && make build && docker compose up -d --build && ./build/mdmlab prepare db --dev && ./build/mdmlab serve --dev --dev_license --dev_expired_license
```

PS. допишу compose чтобы можно одной кнопкой было поднять, а не вот это вот все



Комада для запуска osquery 

```sh 
go run ./orbit/cmd/orbit \
    --dev-mode \
    --disable-updates \
    --root-dir /tmp/orbit \
    --mdmlab-url https://localhost:8080 \
    --insecure \
    --enroll-secret {Сюда надо встваить свой enroll-secret} \
    -- --verbose
```
