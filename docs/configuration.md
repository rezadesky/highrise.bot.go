# Configuration & Options

`highrisebot` uses Go **Functional Options** for clean, flexible, and type-safe configuration.

---

## Default Configuration

If no options are provided, `highrisebot.New(token, roomID)` uses sensible production defaults:

```go
bot := highrisebot.New("TOKEN", "ROOM_ID")
```

- **Gateway URL**: `wss://highrise.game/web/botapi`
- **Web API URL**: `https://webapi.highrise.game`
- **Handshake Timeout**: `15s`
- **Write Deadline**: `10s`
- **Pong Wait**: `60s`
- **Max Reconnect Delay**: `30s`
- **Auto Reconnect**: `true`

---

## Customizing Options

You can customize any behavior using option functions:

```go
import (
    "time"
    "github.com/rezadesky/highrisebot"
)

bot := highrisebot.New(
    "YOUR_TOKEN",
    "YOUR_ROOM_ID",
    highrisebot.WithHandshakeTimeout(10 * time.Second),
    highrisebot.WithMaxReconnectDelay(45 * time.Second),
    highrisebot.WithAutoReconnect(true),
)
```

### Available Options Reference

| Option | Parameter | Description |
| :--- | :--- | :--- |
| `WithGatewayURL(url)` | `string` | Override the WebSocket gateway endpoint. |
| `WithWebAPIURL(url)` | `string` | Override the REST Web API base URL. |
| `WithHandshakeTimeout(d)` | `time.Duration` | Maximum time allowed to establish WebSocket TLS handshake. |
| `WithMaxReconnectDelay(d)` | `time.Duration` | The upper bound ceiling for exponential reconnect backoff. |
| `WithAutoReconnect(bool)` | `bool` | Enable or disable automatic reconnection on socket drop. |

---

## Validating Configuration

The client automatically validates your credentials before initiating any socket connection:
- If `Token` is missing, `ErrMissingToken` is returned.
- If `RoomID` is missing, `ErrMissingRoomID` is returned.
