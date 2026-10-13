# Метаданные страницы

Консольный скрипт на Python 3 без внешних зависимостей.

```bash
python3 meta.py https://example.com
```

Вывод:

```text
Title: Example Domain
Meta description: (не найден)
```

Скрипт следует HTTP-перенаправлениям, ждёт ответа до 20 секунд и выводит ошибки в stderr с кодом завершения 1. Если тег отсутствует или пустой, выводится `(не найден)`.

Извлекаются `<title>` и `<meta name="description" content="...">` из HTML, полученного от сервера. JavaScript не выполняется.

При ошибке `CERTIFICATE_VERIFY_FAILED` установите дополнительный набор доверенных сертификатов и повторите запуск:

```bash
python3 -m pip install certifi
python3 meta.py https://example.com
```

Скрипт использует сертификаты Python, системный набор `/etc/ssl/cert.pem` на macOS и `certifi`, если он установлен. Проверка HTTPS остаётся включённой. Для собственного или корпоративного CA можно явно указать PEM-файл:

```bash
python3 meta.py https://example.com --ca-file /path/to/ca.pem
```
