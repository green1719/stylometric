# Stylometry

`Stylometry` — учебный веб-сервис для хранения текстовых образцов и определения простых стилометрических характеристик текста. Карточка образца содержит автора, текстовый отрывок, изображение, короткое видео, статус, создателя и вычисляемые показатели. Для лабораторной работы №3 интерфейс HTML-страниц отключён, а данные доступны через REST API.

## Запуск

```bash
./stylometry_open.sh
```

REST API: `http://127.0.0.1:8080/api`.

- Adminer: `http://127.0.0.1:8082`;
- MinIO: `http://127.0.0.1:9001`;
- bucket MinIO: `stylometry`.

Параметр `ENABLE_FRONTEND=false` отключает маршруты HTML-страниц и статические ресурсы интерфейса. При таком значении `/`, `/samples/feed`, `/samples/draft` и `/samples/grid` возвращают `404 Not Found`, а `/api/...` продолжает работать. Чтобы вернуть интерфейс после защиты, нужно установить `ENABLE_FRONTEND=true` и перезапустить приложение.

## Фиксированный пользователь

Авторизация в этой лабораторной работе не реализована. Функция-singleton `CurrentUser()` из модуля `stylometry_session` создаёт текущего пользователя с `ID = 1`. Этот ID используется при создании и публикации черновика, лайке и мягком удалении.

## Структура базы данных

### Users

| Поле | Тип PostgreSQL | Ключи и назначение |
| --- | --- | --- |
| `ID` | `bigint` | PK, идентификатор пользователя |
| `Name` | `varchar(120)` | имя пользователя, обязательное поле |
| `Password` | `varchar(255)` | пароль, обязательное поле |

### StylometrySamples

| Поле | Тип PostgreSQL | Ключи и назначение |
| --- | --- | --- |
| `ID` | `bigint` | PK, идентификатор текстового образца |
| `TextCreator` | `varchar(160)` | автор текста, обязательное поле |
| `ShortDescription` | `varchar(800)` | текстовый отрывок |
| `Status` | `varchar(20)` | `draft`, `published` или `deleted` |
| `ImageURL` | `varchar(500)` | URL изображения в MinIO |
| `VideoURL` | `varchar(500)` | URL видео в MinIO |
| `AverageSentenceLength` | `numeric` | средняя длина предложения |
| `ConjunctionFrequency` | `numeric` | частота союзов |
| `CreatedAt` | `timestamp with time zone` | дата создания |
| `UpdatedAt` | `timestamp with time zone` | дата публикации или изменения |
| `CreatorID` | `bigint` | FK → `Users.ID`, создатель карточки |

### Likes

| Поле | Тип PostgreSQL | Ключи и назначение |
| --- | --- | --- |
| `ID` | `bigint` | PK, идентификатор лайка |
| `UserID` | `bigint` | FK → `Users.ID` |
| `SampleID` | `bigint` | FK → `StylometrySamples.ID` |

Связи используют `RESTRICT`, поэтому каскадное удаление запрещено. Уникальный индекс `idx_one_draft_per_user` допускает один черновик на пользователя, а `idx_user_sample_like` не допускает повторный лайк одной карточке.

## Формат карточки в ответах

В ответах не передаётся внутренний статус `Status`. Карточка имеет следующие поля:

```json
{
  "stylometry_sample_id": 1,
  "stylometry_sample_text_creator": "Лев Толстой",
  "stylometry_sample_descr": "Все смешалось в доме Облонских.",
  "stylometry_sample_image_url": "http://localhost:9000/stylometry/stylometry_images/stylometry_tolstoy.jpg",
  "stylometry_sample_video_url": "http://localhost:9000/stylometry/stylometry_videos/stylometry_tolstoy.mp4",
  "stylometry_sample_average_sentence_length": 15,
  "stylometry_sample_conjunction_frequency": 10,
  "stylometry_sample_likes_count": 3,
  "stylometry_sample_is_current_user_creator": 1,
  "is_liked": 0,
  "stylometry_sample_created_at": "2026-10-19T12:00:00Z",
  "stylometry_sample_updated_at": "2026-10-19T12:01:00Z"
}
```

`stylometry_sample_is_current_user_creator` показывает, создана ли карточка текущим пользователем. `is_liked` показывает существование записи в `Likes` для текущего пользователя и карточки: `1` — лайк есть, `0` — лайка нет.

## REST API для показа ЛР-3

Базовый адрес: `http://127.0.0.1:8080`. Ниже перечислены десять запросов Postman в порядке демонстрации.

### 01. Список опубликованных образцов

- **Метод и URL:** `GET /api/samples`
- **Назначение:** возвращает только опубликованные образцы; `deleted` не выдаются.
- **Параметры и тело:** не требуются.
- **Коды:** `200 OK`; `500 Internal Server Error` при ошибке БД.

```json
{ "items": [{ "stylometry_sample_id": 1, "stylometry_sample_text_creator": "Лев Толстой", "is_liked": 0 }] }
```

### 02. Список образцов с фильтром

- **Метод и URL:** `GET /api/samples?stylometry_avg_len=15`
- **Назначение:** возвращает опубликованные образцы с `AverageSentenceLength` не больше переданного значения.
- **Параметры:** `stylometry_avg_len` — число от `0` до `30`.
- **Тело:** не требуется.
- **Коды:** `200 OK`; `400 Bad Request`, если параметр не является числом или выходит за диапазон; `500 Internal Server Error`.

```json
{ "items": [{ "stylometry_sample_id": 1, "stylometry_sample_average_sentence_length": 15, "is_liked": 0 }] }
```

Фильтр выполняется в GORM-запросе как `AverageSentenceLength <= stylometry_avg_len`, а не над массивом в Go.

### 03. Создание черновика с медиа

- **Метод и URL:** `POST /api/samples`
- **Назначение:** создаёт или обновляет один черновик текущего пользователя.
- **Тип тела:** `multipart/form-data`.
- **Поля:** `image` — обязательный файл изображения; `video` — обязательный файл короткого видео; `text` — обязательное многострочное поле.

Первая строка `text` — имя автора, все последующие строки — текстовый отрывок. Поддерживаются `\n` и `\r\n`; пробелы вокруг имени автора удаляются.

```text
text=Лев Толстой
Все смешалось в доме Облонских.
Жена узнала, что муж был в связи с бывшею в их доме гувернанткой.
```

- **Коды:** `201 Created`; `400 Bad Request`, если отсутствует файл, автор или отрывок; `500 Internal Server Error` при ошибке MinIO или БД.

```json
{
  "draft": {
    "stylometry_sample_id": 8,
    "stylometry_sample_text_creator": "Лев Толстой",
    "stylometry_sample_descr": "Все смешалось в доме Облонских.\nЖена узнала, что муж был в связи с бывшею в их доме гувернанткой.",
    "is_liked": 0
  }
}
```

Файлы загружаются в MinIO; в БД сохраняются их URL. Отдельного поля автора нет.

### 04. Получение черновика текущего пользователя

- **Метод и URL:** `GET /api/samples/draft`
- **Назначение:** получает единственный черновик фиксированного текущего пользователя.
- **Параметры и тело:** не требуются.
- **Коды:** `200 OK`; `500 Internal Server Error`.

```json
{ "has_draft": true, "draft": { "stylometry_sample_id": 8, "stylometry_sample_text_creator": "Лев Толстой", "is_liked": 0 } }
```

Если черновика нет, сервер возвращает `{"has_draft": false, "draft": null}` с кодом `200 OK`.

### 05. Публикация черновика

- **Метод и URL:** `PUT /api/samples/draft/publish`
- **Назначение:** меняет статус черновика текущего пользователя на `published`.
- **Параметры и тело:** не требуются.
- **Коды:** `200 OK`; `404 Not Found`, если черновика нет; `500 Internal Server Error`.

```json
{
  "published": true,
  "sample": {
    "stylometry_sample_id": 8,
    "stylometry_sample_average_sentence_length": 12.5,
    "stylometry_sample_conjunction_frequency": 4.2,
    "is_liked": 0
  }
}
```

Статус, `UpdatedAt` и стилометрические показатели рассчитываются на сервере и не приходят от клиента.

### 06. Первый элемент ленты

- **Метод и URL:** `GET /api/samples/feed`
- **Назначение:** получает первый опубликованный образец по возрастанию `ID`.
- **Параметры и тело:** не требуются.
- **Коды:** `200 OK`; `404 Not Found`, если опубликованных карточек нет; `500 Internal Server Error`.

```json
{ "stylometry_sample_id": 1, "stylometry_sample_text_creator": "Лев Толстой", "is_liked": 0 }
```

Возвращается один объект, а не массив. Метод продолжает работать при отключённом фронтенде.

### 07. Следующий элемент ленты

- **Метод и URL:** `GET /api/samples/feed?stylometry_id=1&next=true`
- **Назначение:** получает следующую опубликованную карточку.
- **Параметры:** `stylometry_id` — ID текущей карточки; `next=true` — переход вперёд.
- **Тело:** не требуется.
- **Коды:** `200 OK`; `400 Bad Request` при некорректном ID; `404 Not Found`, если лента пуста; `500 Internal Server Error`.

```json
{ "stylometry_sample_id": 2, "stylometry_sample_text_creator": "Антон Чехов", "is_liked": 0 }
```

Если следующей записи нет, репозиторий снова получает первую опубликованную карточку: лента циклическая.

### 08. Проверка результата лайка

- **Метод и URL:** `GET /api/samples/feed?stylometry_id=1`
- **Назначение:** получает указанную опубликованную карточку и фактическое состояние лайка текущего пользователя.
- **Параметры:** `stylometry_id` — ID карточки.
- **Тело:** не требуется.
- **Коды:** `200 OK`; `400 Bad Request` при некорректном ID; `404 Not Found`, если карточка не опубликована или не найдена; `500 Internal Server Error`.

```json
{ "stylometry_sample_id": 1, "stylometry_sample_likes_count": 4, "is_liked": 1 }
```

`is_liked` вычисляется по таблице `Likes`: после постановки лайка возвращается `1`, после снятия — `0`.

### 09. Мягкое удаление образца

- **Метод и URL:** `DELETE /api/samples/8`
- **Назначение:** помечает карточку текущего пользователя как удалённую.
- **Параметры:** `id` в URL — ID карточки.
- **Тело:** не требуется.
- **Коды:** `200 OK`; `400 Bad Request` при некорректном ID; `404 Not Found`, если карточка не принадлежит пользователю, отсутствует или уже удалена.

```json
{ "deleted": true }
```

Метод использует явный курсор PostgreSQL: `DECLARE`, `FETCH`, `UPDATE`, `CLOSE`, `COMMIT`. Строка остаётся в `StylometrySamples`, но получает статус `deleted`.

### 10. Регистрация пользователя

- **Метод и URL:** `POST /api/users`
- **Назначение:** создаёт пользователя через GORM.
- **Тело:** JSON или form-data с именем и паролем.

```json
{ "stylometry_sample_name": "postman_user", "stylometry_sample_password": "stylometry123" }
```

- **Коды:** `201 Created`; `400 Bad Request`, если имя отсутствует; `409 Conflict`, если имя уже занято; `500 Internal Server Error`.

```json
{ "user": { "stylometry_sample_id": 4, "stylometry_sample_name": "postman_user" } }
```

Пароль не возвращается. Если пароль не передан, сервер использует `stylometry123`.

## Вспомогательный метод постановки лайка

Перед проверкой №08 состояние `is_liked` меняет реальный маршрут `POST /api/samples/:id/like`. Тело `{"stylometry_like": 1}` ставит лайк, а `{"stylometry_like": 0}` снимает его. Ответ — `200 OK` с `liked` и обновлённой карточкой. Этот технический маршрут не входит в порядок десяти скриншотов, но необходим для показа `is_liked: 0` и `is_liked: 1`.

## Заглушки следующей лабораторной

`POST /api/auth/login` и `POST /api/auth/logout` сохранены как заглушки для лабораторной работы №4 и не входят в десять запросов ЛР-3.

## Миграция имён

При запуске приложение проверяет старую схему и переименовывает таблицы и столбцы через `ALTER TABLE ... RENAME` и `ALTER TABLE ... RENAME COLUMN`. Одноразовый SQL-вариант находится в `migrations/001_rename_stylometry_schema.sql`; данные, первичные ключи и внешние ключи сохраняются.
