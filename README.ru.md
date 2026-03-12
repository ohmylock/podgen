<p align="right">
  <a href="README.md">Read in English</a>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/go-1.25+-00ADD8?style=flat&logo=go" alt="Go версия 1.25 и выше">
  <a href="https://github.com/ohmylock/podgen/actions/workflows/ci.yml"><img src="https://github.com/ohmylock/podgen/actions/workflows/ci.yml/badge.svg" alt="Статус CI сборки"></a>
  <a href="https://github.com/ohmylock/podgen/releases"><img src="https://img.shields.io/github/v/release/ohmylock/podgen?include_prereleases" alt="Последний релиз"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="Лицензия MIT"></a>
  <a href="https://goreportcard.com/report/github.com/ohmylock/podgen"><img src="https://goreportcard.com/badge/github.com/ohmylock/podgen" alt="Go Report Card"></a>
</p>

<h1 align="center">podgen</h1>

<p align="center">
  <b>Генератор подкастов — загрузка эпизодов в S3 и создание RSS-лент</b><br>
  <i>CLI-инструмент для управления подкастами: сканирование MP3, извлечение метаданных, генерация обложек</i>
</p>

<p align="center">
  <a href="#быстрый-старт">Быстрый старт</a> •
  <a href="#возможности">Возможности</a> •
  <a href="#установка">Установка</a> •
  <a href="#использование">Использование</a> •
  <a href="#конфигурация">Конфигурация</a>
</p>

---

## Что такое podgen?

**podgen** — утилита командной строки для управления подкастами. Загружает MP3-эпизоды в S3-совместимое хранилище и генерирует RSS-ленты, совместимые с Apple Podcasts, Spotify и другими подкаст-плеерами.

Вместо ручной загрузки файлов и редактирования XML, podgen автоматизирует весь процесс:
- **Сканирование** — находит новые MP3-файлы в папках подкастов
- **Метаданные** — извлекает ID3-теги (название, исполнитель, длительность)
- **Загрузка** — параллельная загрузка в S3 с прогресс-баром
- **RSS-лента** — генерация валидной ленты с itunes-тегами
- **Обложки** — автоматическая генерация artwork для новых подкастов

## Возможности

- **Загрузка в S3** — любое S3-совместимое хранилище (AWS, Minio, Yandex Cloud и др.)
- **RSS/Atom лента** — совместима с Apple Podcasts, Spotify, Google Podcasts
- **Извлечение метаданных** — ID3v2 теги: название, исполнитель, альбом, год, длительность
- **Генерация обложек** — 3000x3000 PNG с градиентами в разных стилях
- **Прогресс-бар** — визуальное отображение загрузки в терминале
- **Откат** — отмена последней загрузки или конкретной сессии
- **Множество подкастов** — один конфиг для нескольких подкастов
- **Graceful shutdown** — корректное завершение по Ctrl+C

## Быстрый старт

```bash
# Установка
brew install ohmylock/tools/podgen

# Создать конфиг
cat > podgen.yml << 'EOF'
podcasts:
  mypodcast:
    title: "Мой Подкаст"
    folder: "mypodcast"
    info:
      author: "Автор"
      email: "author@example.com"
      category: "Technology"

storage:
  folder: "episodes"

cloud_storage:
  endpoint_url: "s3.amazonaws.com"
  bucket: "my-podcast-bucket"
  region: "us-east-1"
  secrets:
    aws_key: ${AWS_ACCESS_KEY_ID}
    aws_secret: ${AWS_SECRET_ACCESS_KEY}
EOF

# Сканировать и загрузить
podgen -s -u -p mypodcast
```

## Установка

### Homebrew (macOS/Linux)

```bash
brew install ohmylock/tools/podgen
```

### Скачать бинарник

Скачайте подходящий бинарник для вашей платформы со страницы [releases](https://github.com/ohmylock/podgen/releases):

| Платформа | Архитектура | Файл |
|----------|--------------|------|
| macOS | Apple Silicon | `podgen_*_darwin_arm64.tar.gz` |
| macOS | Intel | `podgen_*_darwin_amd64.tar.gz` |
| Linux | x86_64 | `podgen_*_linux_amd64.tar.gz` |
| Linux | ARM64 | `podgen_*_linux_arm64.tar.gz` |
| Windows | x86_64 | `podgen_*_windows_amd64.zip` |

### Go install

```bash
go install github.com/ohmylock/podgen/cmd/podgen@latest
```

### Сборка из исходников

```bash
git clone https://github.com/ohmylock/podgen.git
cd podgen
make build
# бинарник: bin/podgen
```

## Использование

### Основные команды

```bash
# Сканировать новые эпизоды + загрузить
podgen -s -u -p mypodcast

# Все подкасты из конфига
podgen -s -u -a

# Только сканировать (без загрузки)
podgen -s -p mypodcast

# Только загрузить (уже отсканированные)
podgen -u -p mypodcast

# Перегенерировать RSS-ленту
podgen -f -p mypodcast

# Загрузить обложку
podgen -i -p mypodcast
```

### Управление подкастами

```bash
# Добавить новый подкаст из папки
podgen --add myfolder --title "Название подкаста"

# Показать RSS URL
podgen --rss -p mypodcast
```

### Откат

```bash
# Откатить последнюю загрузку
podgen -r -p mypodcast

# Откатить конкретную сессию
podgen --rollback-session=session-id -p mypodcast
```

### Генерация обложек

```bash
# Сгенерировать обложку (по умолчанию aurora стиль)
podgen -g -p mypodcast

# С конкретным стилем
podgen -g -p mypodcast --artwork-style=letter
```

#### Доступные стили обложек

| Стиль | Описание |
|-------|----------|
| `aurora` | Mesh-градиент с яркими цветами, стиль северного сияния (по умолчанию) |
| `letter` | Большая первая буква + мягкий градиент |
| `solid` | Сплошной пастельный цвет |
| `gradient` | Вертикальный градиент |
| `gradient-diagonal` | Диагональный градиент |
| `radial` | Радиальный градиент (центр светлее) |
| `circles` | Полупрозрачные круги на пастельном фоне |
| `blobs` | Органические формы на пастельном фоне |
| `noise` | Градиент с текстурой шума |

## Конфигурация

### Расположение файлов

podgen ищет конфиг в следующем порядке:

1. Флаг `--conf` или переменная окружения `PODGEN_CONF`
2. `~/.config/podgen/config.yaml` (рекомендуется)
3. `./podgen.yml` (текущая директория, для совместимости)
4. `./configs/podgen.yml` (для совместимости)

Рекомендуемое расположение — `~/.config/podgen/config.yaml`.

### Расположение базы данных

По умолчанию база данных хранится в `~/.config/podgen/podgen.db`. Можно переопределить:
- `database.path` в конфиге
- Флаг `-d` / `--db`
- Переменная окружения `PODGEN_DB`

### Формат конфига

```yaml
podcasts:
  mypodcast:
    title: "Мой Подкаст"
    folder: "mypodcast"           # папка с MP3 файлами
    max_size: 10000000            # макс. размер загрузки за раз (опционально)
    delete_old_episodes: true     # удалять старые эпизоды перед загрузкой
    info:
      author: "Имя Автора"
      owner: "Владелец"
      email: "email@example.com"
      category: "Technology"      # категория Apple Podcasts
      language: "ru"              # язык RSS-ленты

database:
  type: "sqlite"                  # sqlite (по умолчанию) или bolt
  path: "podgen.db"

storage:
  folder: "episodes"              # локальная папка с MP3 файлами

upload:
  chunk_size: 3                   # параллельная загрузка

artwork:
  auto_generate: true             # автогенерация обложек

cloud_storage:
  endpoint_url: "s3.amazonaws.com"
  bucket: "my-bucket"
  region: "us-east-1"
  secrets:
    aws_key: "AKIAIOSFODNN7EXAMPLE"
    aws_secret: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
```

### Переменные окружения

| Переменная | Описание |
|------------|----------|
| `PODGEN_CONF` | Путь к конфиг-файлу (по умолчанию: `~/.config/podgen/config.yaml`) |
| `PODGEN_DB` | Путь к базе данных (по умолчанию: `~/.config/podgen/podgen.db`) |

```bash
PODGEN_CONF=/etc/podgen/config.yml PODGEN_DB=/var/lib/podgen/data.sqlite podgen -s -u -a
```

## Хранилища данных

podgen поддерживает несколько бэкендов для хранения метаданных эпизодов:

- **SQLite** (по умолчанию) — рекомендуется для большинства пользователей
- **BoltDB** — устаревший формат, поддерживается для совместимости

### Миграция между бэкендами

```bash
# Миграция из BoltDB в SQLite
podgen --migrate-from=bolt:/path/to/podgen.bdb -d /path/to/podgen.sqlite
```

## Извлечение метаданных

При сканировании MP3-файлов podgen автоматически читает ID3v2 теги:

- **Title** — название эпизода (fallback: имя файла)
- **Artist, Album, Year** — комбинируются в описание
- **Comment** — добавляется к описанию
- **Duration** — тег `itunes:duration`
- **Year/Date** — дата публикации (fallback: дата из имени файла YYYY-MM-DD)

## Справочник опций

```
Опции приложения:
  -c, --conf=             путь к конфигу (по умолчанию: ~/.config/podgen/config.yaml)
  -d, --db=               путь к базе данных (по умолчанию: ~/.config/podgen/podgen.db)
  -s, --scan              найти и добавить новые эпизоды
  -u, --upload            загрузить эпизоды
  -f, --feed              перегенерировать RSS-ленты
  -i, --image             загрузить обложку подкаста
  -p, --podcast=          имена подкастов (через запятую)
  -a, --all               все подкасты
  -r, --rollback          откатить последний эпизод
      --rollback-session= откатить по имени сессии
      --rss               показать URL RSS-ленты
      --migrate-from=     миграция из другой БД (формат: type:path)
      --add-podcast=      добавить подкаст из папки
      --add=              алиас для --add-podcast
      --title=            название подкаста (с --add-podcast)
      --clear             удалить старые эпизоды перед загрузкой
  -g, --generate-artwork  (пере)генерировать обложку
      --artwork-style=    стиль обложки (aurora, letter, solid, gradient...)

Опции справки:
  -h, --help              показать эту справку
```

## Makefile

```bash
make build          # скомпилировать bin/podgen
make test           # запустить тесты с race detector
make cover          # тесты с покрытием
make lint           # golangci-lint
make fmt            # форматирование кода
make install        # установить в /usr/local/bin
make release        # goreleaser release
make release-check  # goreleaser dry-run
make clean          # удалить артефакты
```

## Вклад в проект

Вклад приветствуется! Прочитайте [руководство по участию](CONTRIBUTING.md) перед отправкой PR.

## Лицензия

Лицензия MIT — см. файл [LICENSE](LICENSE).

---

<p align="center">
  Сделано с любовью для подкастеров
</p>
