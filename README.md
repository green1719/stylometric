# stylometry

Проект для лабораторных работ по теме стилометрии. Приложение хранит текстовые образцы разных авторов, считает простые стилометрические признаки и отдаёт данные через веб-страницы и REST API.

Основная сущность проекта — текстовый образец. В моей теме это аналог услуги из задания: карточка содержит автора текста, текстовый отрывок, изображение, видео, статус, создателя и рассчитанные признаки.

## Запуск

```bash
./stylometry_open.sh
```

После запуска:

- приложение: `http://127.0.0.1:8080/samples/feed`
- Adminer: `http://127.0.0.1:8082`
- MinIO: `http://127.0.0.1:9001`

Данные для Adminer:

- System: `PostgreSQL`
- Server: `postgres`
- Username: `stylometry_user`
- Password: `stylometry_password`
- Database: `stylometry_db`

Данные для MinIO:

- login: `root`
- password: `rootpassword`
- bucket: `stylometry`

## Текущий пользователь

В лабораторной работе авторизация ещё не реализована полностью, поэтому текущий пользователь зафиксирован на бэкенде. Singleton-функция `CurrentUser()` находится в модуле `stylometry_session` и возвращает пользователя с id `1`.

Этот id используется при создании черновика, получении черновика, публикации, лайке и удалении карточки.

## HTTP API

Все методы API начинаются с `/api`.

### GET /api/samples

Возвращает список опубликованных текстовых образцов. Записи со статусом `deleted` клиенту не передаются.

Фильтр:

```text
/api/samples?stylometry_avg_len=10
```

Параметр `stylometry_avg_len` ограничивает максимальную среднюю длину предложения.

### GET /api/samples/feed

Возвращает один опубликованный образец для ленты. Без параметров возвращается первый опубликованный образец.

Для перехода к следующему элементу:

```text
/api/samples/feed?stylometry_id=1&next=true
```

Если следующего элемента нет, сервер возвращает первый опубликованный образец.

### GET /api/samples/draft

Возвращает черновик текущего пользователя. Id в URL не передаётся, потому что черновик определяется по текущему пользователю.

У одного пользователя может быть только один черновик.

### POST /api/samples

Создаёт или обновляет черновик текстового образца. Запрос отправляется как `multipart/form-data`.

Поля:

- `stylometry_text_creator` — автор текста;
- `stylometry_sample_descr` — текстовый отрывок;
- `stylometry_image` — файл изображения;
- `stylometry_video` — файл короткого видео.

Файлы загружаются в MinIO. В БД сохраняются ссылки на загруженные объекты.

### PUT /api/samples/draft/publish

Публикует черновик текущего пользователя. Тело запроса не требуется.

На бэкенде меняется статус на `published`, рассчитываются стилометрические признаки и заполняется поле `StylometrySampleUpdatedAt`.

### DELETE /api/samples/:id

Логически удаляет текстовый образец текущего пользователя.

Физически запись из БД не удаляется. В таблице меняется статус на `deleted`. Удаление сделано через SQL-запрос `UPDATE`, без ORM-модели.

### POST /api/samples/:id/like

Ставит или снимает лайк от текущего пользователя.

Тело запроса:

```json
{
  "stylometry_like": 1
}
```

`1` ставит лайк, `0` снимает лайк. Повторная постановка лайка не создаёт дубликат.

### POST /api/users

Регистрирует нового пользователя.

Тело запроса:

```json
{
  "stylometry_sample_name": "Ivan Petrov",
  "stylometry_sample_password": "stylometry123"
}
```

В ответе пароль не возвращается.

### POST /api/auth/login

Заглушка для следующей лабораторной работы. Возвращает успешную аутентификацию текущего фиксированного пользователя.

### POST /api/auth/logout

Заглушка для следующей лабораторной работы. Возвращает успешную деавторизацию.

## Таблицы БД

### StylometrySampleUsers

Пользователи приложения.

| Поле | Описание |
| --- | --- |
| `StylometrySampleID` | первичный ключ |
| `StylometrySampleName` | имя пользователя |
| `StylometrySamplePassword` | пароль |

### StylometrySampleTextSamples

Текстовые образцы авторов.

| Поле | Описание |
| --- | --- |
| `StylometrySampleID` | первичный ключ |
| `StylometrySampleTextCreator` | автор текста |
| `StylometrySampleShortDescription` | текстовый отрывок |
| `StylometrySampleStatus` | статус: `draft`, `published`, `deleted` |
| `StylometrySampleImageURL` | ссылка на изображение в MinIO |
| `StylometrySampleVideoURL` | ссылка на видео в MinIO |
| `StylometrySampleAverageSentenceLength` | средняя длина предложения |
| `StylometrySampleConjunctionFrequency` | частота союзов |
| `StylometrySampleCreatedAt` | дата создания |
| `StylometrySampleUpdatedAt` | дата публикации/обновления |
| `StylometrySampleCreatorID` | внешний ключ на пользователя |

### StylometrySampleLikes

Лайки текстовых образцов.

| Поле | Описание |
| --- | --- |
| `StylometrySampleLikeID` | первичный ключ |
| `StylometrySampleUserID` | внешний ключ на пользователя |
| `StylometrySampleTextSampleID` | внешний ключ на текстовый образец |

Пара `StylometrySampleUserID` и `StylometrySampleTextSampleID` уникальна, поэтому один пользователь не может поставить два одинаковых лайка одному образцу.

## Модели и сериализаторы

Модели GORM находятся в файле:

```text
stylometry_internal/stylometry_app/stylometry_model/stylometry_model.go
```

Сериализаторы ответов API находятся в файле:

```text
stylometry_internal/stylometry_app/stylometry_serializer/stylometry_serializer.go
```

В ответах API статус карточки не выводится. Он используется на бэкенде для бизнес-логики.

## UML

UML-диаграмма проекта оформляется отдельно в StarUML. В ней должны быть показаны домены API, модели, таблицы БД и страницы фронтенда.
