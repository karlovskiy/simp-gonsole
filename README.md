Simp console client
================================================

Installation
=====

Archlinux
---------
You can install from the [Arch User Repository (AUR)](https://aur.archlinux.org/packages/simp-gonsole-git/)

Source
------
```bash
$ go build -o simp-gonsole . 
```

Usage
=====

```bash
simp-gonsole <server address> <nickname>
```

Arguments
=========

- server address - simp server host and port in format: host:port, for example: localhost:7777
- nickname - your chat nickname

Flags
=====

Enable debug logging. Default: false
----------------------------------
```bash
simp-gonsole -d <server address> <nickname>   
```