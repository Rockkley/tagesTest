Запустить сервер - go run cmd/server/main.go

Запустить клиент - go run cmd/client/client.go

Методы клиента:
* uploadFile загружает файл в директорию file_storage (по умолчанию)
* downloadFile загружает файл из директории file_storage в директорию downloads

Методы сервера:
* downloadFile загружает файл из files в downloads, добавляя префикс downloaded_ в название файла
* listFiles возвращает список файлов в директории file_storage в формате - Имя файла | Дата создания | Дата обновления
