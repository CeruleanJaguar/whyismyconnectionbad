# whyismyconnectionbad
Network util that does a ping/packet loss test against your router, and sites that you specify.

Requires Super user priveleges as it uses priviledged ICMP.

Uses go v1.11 modules.

## Building
```bash
go mod download
go build
```

## Quick run
```bash
sudo go mod download
sudo go run main.go
```

## Usage
```bash
whyismyconnectionbad [--help] [addrs...]
```

Addresses should not have a protocol prefix.
