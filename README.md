# mqtt-driver

> **Driver de integração MQTT → SQL Server** — consome mensagens de um broker MQTT de campo (*raw data*), normaliza os timestamps para UTC e republica os dados formatados em um segundo tópico MQTT.

---

## Sumário

- [Visão Geral](#visão-geral)
- [Arquitetura](#arquitetura)
- [Fluxo de Dados](#fluxo-de-dados)
- [Estrutura de Pastas](#estrutura-de-pastas)
- [Pré-requisitos](#pré-requisitos)
- [Configuração](#configuração)
- [Como Executar](#como-executar)
  - [Localmente](#localmente)
  - [Docker](#docker)
- [Tópicos MQTT](#tópicos-mqtt)
- [Banco de Dados](#banco-de-dados)
- [Dependências](#dependências)

---

## Visão Geral

O `mqtt-driver` é um serviço escrito em **Go** que atua como **middleware** entre dispositivos de campo e sistemas superiores de supervisão. Ele:

1. Consulta um banco **SQL Server** para descobrir quais tags/tópicos precisam ser monitorados.
2. Assina os tópicos *raw* no broker MQTT de origem (dados vindos de dispositivos com protocolo `mqtt`).
3. Converte o timestamp recebido em **UTC-3 (horário de Brasília)** para **UTC**.
4. Publica o payload normalizado em um tópico de destino no mesmo broker.

---

## Arquitetura

```
┌─────────────────────────────────────────────────────────┐
│                       mqtt-driver                       │
│                                                         │
│  ┌──────────┐   tags    ┌──────────┐                    │
│  │ SQL Server│─────────▶│ Subscriber│                   │
│  │(mqtt_tags)│          │(N goroutines) │               │
│  └──────────┘          └─────┬─────┘                    │
│                              │ raw payload              │
│                              ▼                          │
│                      ┌──────────────┐                   │
│                      │ handleMessage│                   │
│                      │ (UTC convert)│                   │
│                      └──────┬───────┘                   │
│                             │ formatted payload         │
│                             ▼                           │
│                      ┌──────────┐                       │
│                      │ Publisher│                       │
│                      └──────────┘                       │
└─────────────────────────────────────────────────────────┘
```

O serviço mantém **duas conexões MQTT simultâneas** — uma dedicada exclusivamente para publicação (`-pub`) e outra para subscrição (`-sub`) — garantindo isolamento e evitando conflitos de callback no mesmo `ClientID`.

---

## Fluxo de Dados

```
Dispositivo de campo
        │
        │  JSON: { "TS": "19/03/2026 15:04:05", "Val": "42.5" }
        ▼
/devices/raw_data/erd/{deviceID}/{rawTopic}
        │
        │  [Subscriber] deserializa → converte TS UTC-3 → UTC
        ▼
/devices/formatted_data/erd/{deviceID}/{tagID}
        │
        │  JSON: { "TS": "2026-03-19T18:04:05Z", "Val": "42.5" }
        ▼
    Sistema de supervisão / SCADA
```

---

## Estrutura de Pastas

```
mqtt-driver/
├── cmd/
│   ├── main.go          # Ponto de entrada da aplicação
│   └── .env             # Variáveis de ambiente (não versionar em produção)
│
├── internal/
│   ├── config/
│   │   └── config.go    # Carrega e valida configurações via variáveis de ambiente
│   │
│   ├── database/
│   │   └── db.go        # Conexão com SQL Server e consulta de tags
│   │
│   ├── models/
│   │   └── models.go    # Structs de domínio (MQTTTag, RawPayload, FormattedPayload)
│   │
│   └── mqtt/
│       ├── publisher.go  # Conexão MQTT de publicação e método Publish()
│       └── subscriber.go # Conexão MQTT de subscrição, lógica de grupos e conversão UTC
│
├── Dockerfile            # Build multi-stage (builder Alpine + imagem final Alpine)
├── go.mod                # Módulo Go e dependências
└── go.sum                # Checksums das dependências
```

### `cmd/`

Contém o **ponto de entrada** (`main.go`) e o arquivo de configuração local (`.env`).

- **`main.go`** — Orquestra toda a inicialização do serviço na seguinte ordem:
  1. Carrega configuração (`config.Load`).
  2. Conecta ao SQL Server (`database.Connect`).
  3. Carrega as tags de entrada (`database.LoadTags`).
  4. Instancia o `Publisher` MQTT.
  5. Instancia o `Subscriber` MQTT e assina todos os grupos de tags em paralelo.
  6. Aguarda sinal de SO (`SIGINT` / `SIGTERM`) para *graceful shutdown*.

---

### `internal/config/`

Responsável por **carregar e expor as configurações** da aplicação.

- **`config.go`** — Define a struct `Config` e a função `Load()`:
  - Usa `godotenv` para carregar automaticamente o arquivo `.env` (se existir).
  - Usa variáveis de ambiente do sistema operacional como fonte primária.
  - Fornece valores padrão sensatos para todas as chaves.

---

### `internal/database/`

Responsável pela **conectividade com o SQL Server** e pela leitura das tags a serem monitoradas.

- **`db.go`**
  - `Connect(cfg)` — Abre e valida (via `PingContext`) a conexão com o SQL Server usando o driver `go-mssqldb`.
  - `LoadTags(ctx, db)` — Executa uma query que busca todas as `mqtt_tags` associadas a `devices` com `protocol = 'mqtt'`, retornando uma slice de `models.MQTTTag`.

**Query executada:**
```sql
SELECT
    CAST(mt.id       AS VARCHAR(36)) AS tag_id,
    mt.name                          AS tag_name,
    mt.raw_topic,
    mt.formatted_topic,
    CAST(d.id        AS VARCHAR(36)) AS device_id
FROM mqtt_tags mt
JOIN devices d ON d.id = mt.id_device
WHERE d.protocol = 'mqtt'
ORDER BY d.id, mt.id
```

---

### `internal/models/`

Define as **structs de domínio** compartilhadas entre os pacotes.

- **`models.go`**

| Struct | Descrição |
|---|---|
| `MQTTTag` | Representa uma tag cadastrada no banco (`TagID`, `TagName`, `RawTopic`, `FormattedTopic`, `DeviceID`). |
| `RawPayload` | Formato JSON recebido do dispositivo de campo. Campo `TS` em UTC-3 sem offset explícito. |
| `FormattedPayload` | Formato JSON publicado no tópico de saída. Campo `TS` sempre em UTC (RFC 3339). |

---

### `internal/mqtt/`

Núcleo do serviço. Contém a lógica de **subscrição paralela, processamento de mensagens e publicação**.

#### `publisher.go`

- Cria uma conexão MQTT dedicada para **publicação** com `ClientID = {MQTT_CLIENT_ID}-pub`.
- Habilita reconexão automática (`AutoReconnect`).
- Método `Publish(deviceID, tagID, payload)` — publica no tópico:
  ```
  /devices/formatted_data/erd/{deviceID}/{tagID}
  ```
  com QoS 0 e *retain* ativo (`true`).

#### `subscriber.go`

- Cria uma conexão MQTT dedicada para **subscrição** com `ClientID = {MQTT_CLIENT_ID}-sub`.
- **`SubscribeGroups(tags)`** — divide as tags em grupos de tamanho `TAG_GROUP_SIZE` usando `chunkTags()` e inicia uma **goroutine por grupo**, permitindo subscrições paralelas sem sobrecarga de uma única conexão.
- **`handleMessage(tag, rawBytes)`** — callback invocado para cada mensagem recebida:
  1. Desserializa o JSON do payload.
  2. Converte o timestamp para UTC via `convertToUTC()`.
  3. Serializa o `FormattedPayload` e delega a publicação ao `Publisher`.
- **`convertToUTC(ts)`** — tenta múltiplos formatos de timestamp (com prioridade para o formato brasileiro `dd/MM/yyyy HH:mm:ss`). Em caso de falha, usa `time.Now().UTC()` como fallback.

---

## Pré-requisitos

| Requisito | Versão mínima |
|---|---|
| Go | 1.24 |
| SQL Server | 2016+ |
| Broker MQTT | Qualquer compatível com MQTT 3.1.1 (ex: Mosquitto, EMQX) |
| Docker (opcional) | 20.10+ |

---

## Configuração

Copie o arquivo `.env` de exemplo e ajuste conforme o ambiente:

```bash
cp cmd/.env.example cmd/.env
```

| Variável | Descrição | Padrão |
|---|---|---|
| `DB_SERVER` | Endereço do SQL Server | `localhost` |
| `DB_PORT` | Porta do SQL Server | `1433` |
| `DB_USER` | Usuário do banco | `sa` |
| `DB_PASSWORD` | Senha do banco | _(vazio)_ |
| `DB_NAME` | Nome do banco de dados | _(vazio)_ |
| `MQTT_BROKER` | URL do broker MQTT | `tcp://localhost:1883` |
| `MQTT_CLIENT_ID` | Client ID base do MQTT | `mqtt-driver` |
| `MQTT_USERNAME` | Usuário MQTT (opcional) | _(vazio)_ |
| `MQTT_PASSWORD` | Senha MQTT (opcional) | _(vazio)_ |
| `TAG_GROUP_SIZE` | Qtd. de tags por goroutine de subscrição | `5` |

> **⚠️ Atenção:** Nunca versione o arquivo `.env` com credenciais reais em repositórios públicos.

---

## Como Executar

### Localmente

```bash
# Clone o repositório
git clone https://github.com/seu-usuario/mqtt-driver.git
cd mqtt-driver

# Configure as variáveis de ambiente
cp cmd/.env.example cmd/.env
# Edite cmd/.env com suas credenciais

# Instale as dependências e execute
go mod download
go run ./cmd/main.go
```

### Docker

O `Dockerfile` utiliza build **multi-stage** para gerar uma imagem final mínima baseada em `alpine:3.19`.

```bash
# Build da imagem
docker build -t mqtt-driver .

# Execução (as variáveis de ambiente sobrescrevem o .env)
docker run --rm \
  -e DB_SERVER=192.168.0.12 \
  -e DB_PASSWORD=suasenha \
  -e DB_NAME=hydra \
  -e MQTT_BROKER=tcp://broker.exemplo.com:1883 \
  mqtt-driver
```

---

## Tópicos MQTT

| Direção | Padrão de tópico | Descrição |
|---|---|---|
| **Entrada (subscribe)** | `/devices/raw_data/erd/{deviceID}/{rawTopic}` | Dados brutos vindos dos dispositivos de campo |
| **Saída (publish)** | `/devices/formatted_data/erd/{deviceID}/{tagID}` | Dados normalizados com timestamp em UTC |

**Exemplo de payload recebido:**
```json
{ "TS": "19/03/2026 15:04:05", "Val": "42.5" }
```

**Exemplo de payload publicado:**
```json
{ "TS": "2026-03-19T18:04:05Z", "Val": "42.5" }
```

---

## Banco de Dados

O serviço espera as seguintes tabelas no SQL Server:

### `devices`
| Coluna | Tipo | Descrição |
|---|---|---|
| `id` | `UNIQUEIDENTIFIER` | Identificador único do dispositivo |
| `protocol` | `VARCHAR` | Protocolo do dispositivo — filtra por `'mqtt'` |

### `mqtt_tags`
| Coluna | Tipo | Descrição |
|---|---|---|
| `id` | `UNIQUEIDENTIFIER` | Identificador único da tag |
| `name` | `VARCHAR` | Nome descritivo da tag |
| `raw_topic` | `VARCHAR` | Sufixo do tópico MQTT de entrada |
| `formatted_topic` | `VARCHAR` | Sufixo do tópico MQTT de saída (reservado para uso futuro) |
| `id_device` | `UNIQUEIDENTIFIER` | FK para `devices.id` |

---

## Dependências

| Pacote | Descrição |
|---|---|
| [`paho.mqtt.golang`](https://github.com/eclipse/paho.mqtt.golang) | Cliente MQTT para Go (Eclipse Paho) |
| [`go-mssqldb`](https://github.com/microsoft/go-mssqldb) | Driver SQL Server para Go |
| [`godotenv`](https://github.com/joho/godotenv) | Carregamento de arquivos `.env` |
| [`google/uuid`](https://github.com/google/uuid) | Geração e manipulação de UUIDs |
| [`shopspring/decimal`](https://github.com/shopspring/decimal) | Aritmética decimal de alta precisão |
| [`gorilla/websocket`](https://github.com/gorilla/websocket) | Suporte a WebSocket (usado pelo Paho) |

---

*Documentação gerada com base no código-fonte em `2026-03-19`.*
